package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("resource not found")

type Store interface {
	BeginTx(ctx context.Context, options sql.TxOptions) (Tx, error)

	Chat(ctx context.Context, tx Tx, chatID string) (Chat, error)

	AllMessagesSince(ctx context.Context, tx Tx, t time.Time) ([]Message, error)
	Messages(ctx context.Context, tx Tx, chatID string) ([]Message, error)
	CreateMessage(ctx context.Context, tx Tx, message Message) error
}
