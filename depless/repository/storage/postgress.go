package storage

import (
	"context"
	"database/sql"
)

type SqlStorage struct {
	db DB
}

type DB interface {
	Pipe(ctx context.Context, opts *sql.TxOptions, pipe func(tx *sql.Tx) error) error
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func NewPostgresStorage(db DB) *SqlStorage {
	return &SqlStorage{
		db: db,
	}
}

type Endpoint struct {
}

func (s *SqlStorage) GetEndpoint(ctx context.Context) {

}
