package data

import (
	"context"
	"errors"
)

var ErrTxDone = errors.New("transaction has already been committed or rolled back")

type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
}

type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
}

type Row interface {
	Scan(dest ...any) error
}

type Result interface {
	RowsAffected() (int64, error)
}
