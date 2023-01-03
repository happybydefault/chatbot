package chatbot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

func (c *Client) execTx(
	ctx context.Context,
	options sql.TxOptions,
	fn func(tx data.Tx) error,
) error {
	tx, err := c.store.BeginTx(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to begin data store transaction: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, data.ErrTxDone) {
			c.logger.Error("failed to rollback data store transaction", zap.Error(err))
		}
	}()

	err = fn(tx)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit data store transaction: %w", err)
	}

	return nil
}
