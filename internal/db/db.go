package db

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"sync"
	"time"
)

var (
	jsonFilePath = "connections.json"
	mu           sync.RWMutex
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

// InitDB initializes the local JSON database file.
func InitDB(dbPath string) error {
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
		return os.WriteFile(jsonFilePath, data, 0644)
	}
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

// SaveConnection inserts or updates a connection in the JSON database.
func SaveConnection(conn *Connection) error {
	mu.Lock()
	defer mu.Unlock()

	conns, err := readAllConnections()
	if err != nil {
		return err
	}

	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = time.Now()
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

	return writeAllConnections(conns)
}

// GetConnection retrieves a connection by its ID.
func GetConnection(id string) (*Connection, error) {
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
