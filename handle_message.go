package chatbot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

// TODO: Make this concurrent for multiple chats.
func (c *Client) handleMessage(message *events.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.whatsmeowClient.MarkRead(
		[]types.MessageID{
			message.Info.ID,
		},
		time.Now(),
		message.Info.Chat.ToNonAD(),
		message.Info.Sender.ToNonAD(),
	)
	if err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	if message.Message.GetConversation() == "" {
		c.logger.Debug("ignoring Message event with empty Conversation")
		return nil
	}

	chatID := message.Info.Chat.User

	err = c.execTx(ctx, sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	}, func(tx data.Tx) error {
		_, err := c.store.Chat(ctx, tx, chatID)
		if err != nil {
			return fmt.Errorf("failed to get chat from data store: %w", err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			c.logger.Debug("ignoring Message event from unknown chat")
			return nil
		}
		return fmt.Errorf("failed to execute data store transaction: %w", err)
	}

	err = c.execTx(ctx, sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}, func(tx data.Tx) error {
		err := c.store.CreateMessage(ctx, tx, data.Message{
			ID:           message.Info.ID,
			Conversation: message.Message.GetConversation(),
			ChatID:       chatID,
			SenderID:     message.Info.Sender.User,
		})
		if err != nil {
			return fmt.Errorf("failed to create message from user in data store: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to execute data store transaction: %w", err)
	}

	if c.status == StatusSyncing {
		c.logger.Debug("adding chat to the slice of pending chats", zap.String("chat_id", chatID))
		c.pendingChats[chatID] = struct{}{}
		return nil
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		err = c.handleChat(chatID)
		if err != nil {
			c.logger.Error("failed to handle chat", zap.Error(err))
		}
	}()

	return nil
}
