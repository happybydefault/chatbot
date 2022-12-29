package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/happybydefault/chatbot/data"
)

type Tx struct {
	pgx.Tx
}

func newTx(tx pgx.Tx) *Tx {
	return &Tx{Tx: tx}
}

func (tx Tx) Commit(ctx context.Context) error {
	err := tx.Tx.Commit(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrTxClosed) {
			return fmt.Errorf("%w: %s", data.ErrTxDone, err)
		}
		return err
	}

	return nil
}

func (tx Tx) Rollback(ctx context.Context) error {
	err := tx.Tx.Rollback(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrTxClosed) {
			return fmt.Errorf("%w: %s", data.ErrTxDone, err)
		}
		return err
	}

	return nil
}

func (tx Tx) QueryRow(ctx context.Context, query string, args ...interface{}) data.Row {
	return tx.Tx.QueryRow(ctx, query, args...)
}

func (tx Tx) Query(ctx context.Context, query string, args ...interface{}) (data.Rows, error) {
	rows, err := tx.Tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return newRows(rows), nil
}

func (tx Tx) Exec(ctx context.Context, query string, args ...interface{}) (data.Result, error) {
	commandTag, err := tx.Tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return newResult(commandTag), nil
}
