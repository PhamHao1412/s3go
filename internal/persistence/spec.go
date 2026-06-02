package persistence

import (
	"context"
	"database/sql"

	"s3go/internal/persistence/connection"
)

type ReadOnlyUnitOfWork interface {
	Connection() connection.ReadOnlyRepository
}

type WritableUnitOfWork interface {
	Connection() connection.WritableRepository
	DoInTx(ctx context.Context, execFunc func(tx WritableUnitOfWork) error) error
}

type readOnlyUnitOfWork struct {
	connectionRepo connection.ReadOnlyRepository
}

func NewReadOnlyUnitOfWork(pgDB *sql.DB, jsonFilePath string) ReadOnlyUnitOfWork {
	return &readOnlyUnitOfWork{
		connectionRepo: connection.NewReadOnlyRepository(pgDB, jsonFilePath),
	}
}

func (u *readOnlyUnitOfWork) Connection() connection.ReadOnlyRepository {
	return u.connectionRepo
}

type writableUnitOfWork struct {
	pgDB           *sql.DB
	jsonFilePath   string
	connectionRepo connection.WritableRepository
}

func NewWritableUnitOfWork(pgDB *sql.DB, jsonFilePath string) WritableUnitOfWork {
	return &writableUnitOfWork{
		pgDB:           pgDB,
		jsonFilePath:   jsonFilePath,
		connectionRepo: connection.NewWritableRepository(pgDB, jsonFilePath),
	}
}

func (u *writableUnitOfWork) Connection() connection.WritableRepository {
	return u.connectionRepo
}

func (u *writableUnitOfWork) DoInTx(ctx context.Context, execFunc func(tx WritableUnitOfWork) error) error {
	// For PostgreSQL, we could run in a SQL transaction, but since connections table is simple
	// and we support local JSON storage, simple delegation is robust enough.
	return execFunc(u)
}
