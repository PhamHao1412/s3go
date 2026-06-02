package connection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"sync"
	"time"

	"s3go/internal/entity"
)

var (
	mu sync.RWMutex
)

type repository struct {
	pgDB         *sql.DB
	jsonFilePath string
}

func NewReadOnlyRepository(pgDB *sql.DB, jsonFilePath string) ReadOnlyRepository {
	return &repository{
		pgDB:         pgDB,
		jsonFilePath: jsonFilePath,
	}
}

func NewWritableRepository(pgDB *sql.DB, jsonFilePath string) WritableRepository {
	return &repository{
		pgDB:         pgDB,
		jsonFilePath: jsonFilePath,
	}
}

func (r *repository) isPostgres() bool {
	return r.pgDB != nil
}

func (r *repository) readAllConnections() ([]entity.Connection, error) {
	data, err := os.ReadFile(r.jsonFilePath)
	if err != nil {
		return nil, err
	}

	var conns []entity.Connection
	if err := json.Unmarshal(data, &conns); err != nil {
		return nil, err
	}
	return conns, nil
}

func (r *repository) writeAllConnections(conns []entity.Connection) error {
	data, err := json.MarshalIndent(conns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.jsonFilePath, data, 0644)
}

func (r *repository) GetByID(ctx context.Context, id string) (*entity.Connection, error) {
	if r.isPostgres() {
		query := `SELECT id, name, access_key, secret_key_encrypted, region, bucket, created_at FROM connections WHERE id = $1`
		row := r.pgDB.QueryRowContext(ctx, query, id)
		var conn entity.Connection
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

	conns, err := r.readAllConnections()
	if err != nil {
		return nil, err
	}

	for _, conn := range conns {
		if conn.ID == id {
			c := conn
			return &c, nil
		}
	}

	return nil, errors.New("connection not found")
}

func (r *repository) List(ctx context.Context) ([]*entity.Connection, error) {
	if r.isPostgres() {
		query := `SELECT id, name, access_key, secret_key_encrypted, region, bucket, created_at FROM connections ORDER BY created_at DESC`
		rows, err := r.pgDB.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var result []*entity.Connection
		for rows.Next() {
			var conn entity.Connection
			if err := rows.Scan(&conn.ID, &conn.Name, &conn.AccessKey, &conn.SecretKeyEncrypted, &conn.Region, &conn.Bucket, &conn.CreatedAt); err != nil {
				return nil, err
			}
			result = append(result, &conn)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return result, nil
	}

	mu.RLock()
	defer mu.RUnlock()

	conns, err := r.readAllConnections()
	if err != nil {
		return nil, err
	}

	var result []*entity.Connection
	for _, conn := range conns {
		c := conn
		result = append(result, &c)
	}

	return result, nil
}

func (r *repository) Save(ctx context.Context, conn *entity.Connection) error {
	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = time.Now()
	}

	if r.isPostgres() {
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
		_, err := r.pgDB.ExecContext(ctx, query, conn.ID, conn.Name, conn.AccessKey, conn.SecretKeyEncrypted, conn.Region, conn.Bucket, conn.CreatedAt)
		if err == nil {
			log.Printf("DB: Successfully saved connection '%s' (%s) to PostgreSQL", conn.Name, conn.ID)
		}
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	conns, err := r.readAllConnections()
	if err != nil {
		return err
	}

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

	err = r.writeAllConnections(conns)
	if err == nil {
		log.Printf("DB: Successfully saved connection '%s' (%s) to local JSON file", conn.Name, conn.ID)
	}
	return err
}

func (r *repository) Delete(ctx context.Context, id string) error {
	if r.isPostgres() {
		query := `DELETE FROM connections WHERE id = $1`
		res, err := r.pgDB.ExecContext(ctx, query, id)
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

	conns, err := r.readAllConnections()
	if err != nil {
		return err
	}

	var updatedConns []entity.Connection
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

	return r.writeAllConnections(updatedConns)
}

// EnsureDBFileExists checks if local database file exists, otherwise creates empty list.
func EnsureDBFileExists(jsonFilePath string) error {
	mu.Lock()
	defer mu.Unlock()

	if _, err := os.Stat(jsonFilePath); errors.Is(err, fs.ErrNotExist) {
		emptyList := []entity.Connection{}
		data, err := json.MarshalIndent(emptyList, "", "  ")
		if err != nil {
			return err
		}
		err = os.WriteFile(jsonFilePath, data, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
