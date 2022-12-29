package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

type Store struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewStore(ctx context.Context, connString string, logger *zap.Logger) (*Store, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to construct Postgres connection pool: %w", err)
	}

	return &Store{
		pool:   pool,
		logger: logger,
	}, nil
}

func (s *Store) BeginTx(ctx context.Context, options sql.TxOptions) (data.Tx, error) {
	var isolationLevel pgx.TxIsoLevel
	switch options.Isolation {
	case sql.LevelReadUncommitted:
		isolationLevel = pgx.ReadUncommitted
	case sql.LevelReadCommitted:
		isolationLevel = pgx.ReadCommitted
	case sql.LevelWriteCommitted:
		isolationLevel = pgx.RepeatableRead
	case sql.LevelRepeatableRead:
		isolationLevel = pgx.RepeatableRead
	default:
		isolationLevel = pgx.Serializable
	}

	var accessMode pgx.TxAccessMode
	if options.ReadOnly {
		accessMode = pgx.ReadOnly
	} else {
		accessMode = pgx.ReadWrite
	}

	opts := pgx.TxOptions{
		IsoLevel:       isolationLevel,
		AccessMode:     accessMode,
		DeferrableMode: pgx.NotDeferrable,
	}
	options.ReadOnly = true

	tx, err := s.pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf(": %w", err)
	}

	return newTx(tx), nil
}

func (s *Store) Close() {
	s.pool.Close()
}
