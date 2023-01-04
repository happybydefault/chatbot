package chatbot

import (
	"sync"

	"go.uber.org/zap"
)

func (c *Client) handleOfflineSyncCompleted() error {
	var wg sync.WaitGroup
	wg.Add(len(c.pendingChats))

	for chatID := range c.pendingChats {
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
	c.status = StatusReady

	return nil
}
