package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

var (
	jsonFilePath = "connections.json"
	mu           sync.RWMutex
	pgDB         *sql.DB
)

// Connection represents a saved S3 connection configuration.
type Connection struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	AccessKey          string    `json:"access_key"`
	SecretKeyEncrypted string    `json:"secret_key_encrypted,omitempty"` // Omitted in JSON payloads if empty, handled internally
	Region             string    `json:"region"`
	Bucket             string    `json:"bucket"`
	CreatedAt          time.Time `json:"created_at"`
}

// isPostgres returns true if PostgreSQL backend is active.
func isPostgres() bool {
	return pgDB != nil
}

// InitDB initializes the database storage engine.
func InitDB(dbPath string) error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		// PostgreSQL mode
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			return err
		}

		// Verify connection
		if err := db.Ping(); err != nil {
			db.Close()
			return err
		}

		pgDB = db

		// Automatically run schema migrations to create table if it doesn't exist
		query := `
		CREATE TABLE IF NOT EXISTS connections (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			access_key VARCHAR(255) NOT NULL,
			secret_key_encrypted TEXT NOT NULL,
			region VARCHAR(255) NOT NULL,
			bucket VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL
		);`
		if _, err := pgDB.Exec(query); err != nil {
			pgDB.Close()
			pgDB = nil
			return err
		}

		log.Println("DB: Successfully connected and initialized PostgreSQL database")
		return nil
	}

	// Local JSON Fallback mode
	mu.Lock()
	defer mu.Unlock()

	// Prioritize DB_PATH environment variable for cloud deployments (e.g. Render Persistent Disk)
	if envPath := os.Getenv("DB_PATH"); envPath != "" {
		jsonFilePath = envPath
	} else if dbPath != "" {
		// Fallback to default name
		jsonFilePath = "connections.json"
	}

	// Check if file exists, if not create empty list
	if _, err := os.Stat(jsonFilePath); errors.Is(err, fs.ErrNotExist) {
		emptyList := []Connection{}
		data, err := json.MarshalIndent(emptyList, "", "  ")
		if err != nil {
			return err
		}
		err = os.WriteFile(jsonFilePath, data, 0644)
		if err != nil {
			return err
		}
	}
	log.Printf("DB: Successfully initialized local JSON database at %s", jsonFilePath)
	return nil
}

// readAllConnections reads all connections from the JSON file. (Unsafe, must call under lock)
func readAllConnections() ([]Connection, error) {
	data, err := os.ReadFile(jsonFilePath)
	if err != nil {
		return nil, err
	}

	var conns []Connection
	if err := json.Unmarshal(data, &conns); err != nil {
		return nil, err
	}
	return conns, nil
}

// writeAllConnections writes all connections to the JSON file. (Unsafe, must call under lock)
func writeAllConnections(conns []Connection) error {
	data, err := json.MarshalIndent(conns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(jsonFilePath, data, 0644)
}

// SaveConnection inserts or updates a connection in the database.
func SaveConnection(conn *Connection) error {
	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = time.Now()
	}

	if isPostgres() {
		query := `
		INSERT INTO connections (id, name, access_key, secret_key_encrypted, region, bucket, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			access_key = EXCLUDED.access_key,
			secret_key_encrypted = EXCLUDED.secret_key_encrypted,
			region = EXCLUDED.region,
			bucket = EXCLUDED.bucket,
			created_at = EXCLUDED.created_at;
		`
		_, err := pgDB.Exec(query, conn.ID, conn.Name, conn.AccessKey, conn.SecretKeyEncrypted, conn.Region, conn.Bucket, conn.CreatedAt)
		if err == nil {
			log.Printf("DB: Successfully saved connection '%s' (%s) to PostgreSQL", conn.Name, conn.ID)
		}
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	conns, err := readAllConnections()
	if err != nil {
		return err
	}

	// Check if connection already exists to update it, otherwise append
	updated := false
	for i, c := range conns {
		if c.ID == conn.ID {
			conns[i] = *conn
			updated = true
			break
		}
	}

	if !updated {
		conns = append(conns, *conn)
	}

	err = writeAllConnections(conns)
	if err == nil {
		log.Printf("DB: Successfully saved connection '%s' (%s) to local JSON file", conn.Name, conn.ID)
	}
	return err
}

// GetConnection retrieves a connection by its ID.
func GetConnection(id string) (*Connection, error) {
	if isPostgres() {
		query := `SELECT id, name, access_key, secret_key_encrypted, region, bucket, created_at FROM connections WHERE id = $1`
		row := pgDB.QueryRow(query, id)
		var conn Connection
		err := row.Scan(&conn.ID, &conn.Name, &conn.AccessKey, &conn.SecretKeyEncrypted, &conn.Region, &conn.Bucket, &conn.CreatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.New("connection not found")
			}
			return nil, err
		}
		return &conn, nil
	}

	mu.RLock()
	defer mu.RUnlock()

	conns, err := readAllConnections()
	if err != nil {
		return nil, err
	}

	for _, conn := range conns {
		if conn.ID == id {
			// Return a copy
			c := conn
			return &c, nil
		}
	}

	return nil, errors.New("connection not found")
}

// ListConnections retrieves all connections (excludes sensitive encrypted secret key in JSON responses).
func ListConnections() ([]*Connection, error) {
	if isPostgres() {
		query := `SELECT id, name, access_key, secret_key_encrypted, region, bucket, created_at FROM connections ORDER BY created_at DESC`
		rows, err := pgDB.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var result []*Connection
		for rows.Next() {
			var conn Connection
			if err := rows.Scan(&conn.ID, &conn.Name, &conn.AccessKey, &conn.SecretKeyEncrypted, &conn.Region, &conn.Bucket, &conn.CreatedAt); err != nil {
				return nil, err
			}
			// Strip out SecretKeyEncrypted for security before sending to list endpoint
			conn.SecretKeyEncrypted = ""
			result = append(result, &conn)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return result, nil
	}

	mu.RLock()
	defer mu.RUnlock()

	conns, err := readAllConnections()
	if err != nil {
		return nil, err
	}

	var result []*Connection
	for _, conn := range conns {
		// Create a copy and strip out SecretKeyEncrypted for security before sending to list endpoint
		c := conn
		c.SecretKeyEncrypted = ""
		result = append(result, &c)
	}

	return result, nil
}

// DeleteConnection deletes a connection from the database.
func DeleteConnection(id string) error {
	if isPostgres() {
		query := `DELETE FROM connections WHERE id = $1`
		res, err := pgDB.Exec(query, id)
		if err != nil {
			return err
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return errors.New("connection not found")
		}
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	conns, err := readAllConnections()
	if err != nil {
		return err
	}

	var updatedConns []Connection
	found := false
	for _, conn := range conns {
		if conn.ID == id {
			found = true
			continue
		}
		updatedConns = append(updatedConns, conn)
	}

	if !found {
		return errors.New("connection not found")
	}

	return writeAllConnections(updatedConns)
}
