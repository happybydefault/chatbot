package chatbot

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

func (c *Client) handleOfflineSyncCompletedEvent() error {
	c.messagesWG.Wait()
	c.state = StateSynced

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var pendingMessages []data.Message
	err := c.execTx(ctx, sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	}, func(tx data.Tx) error {
		var err error
		pendingMessages, err = c.store.AllMessagesSince(ctx, tx, c.syncingSince)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.logger.Error("failed to execute data store transaction", zap.Error(err))
	}

	chatIDs := make(map[string]struct{})
	for _, message := range pendingMessages {
		chatIDs[message.ChatID] = struct{}{}
	}

	var wg sync.WaitGroup
	wg.Add(len(chatIDs))
	for chatID := range chatIDs {
		chatID := chatID
		go func() {
			defer wg.Done()

			err := c.handleChat(chatID)
			if err != nil {
				c.logger.Error("failed to handle chat", zap.Error(err))
			}
		}()
	}

	wg.Wait()

	return nil
}
