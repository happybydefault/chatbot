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

func (c *Client) handleMessageEvent(message *events.Message) {
	state := c.state

	c.messagesWG.Add(1)
	go func() {
		defer c.messagesWG.Done()

		err := c.handleMessage(message, state)
		if err != nil {
			c.logger.Error("failed to handle message", zap.Error(err))
		}
	}()
}

func (c *Client) handleMessage(message *events.Message, state State) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	chatID := message.Info.Chat.User

	err := c.execTx(ctx, sql.TxOptions{
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

	if message.Message.GetConversation() == "" {
		err = c.whatsmeowClient.MarkRead(
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

		c.logger.Debug("ignoring Message event with empty Conversation")
		return nil
	}

	err = c.execTx(ctx, sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}, func(tx data.Tx) error {
		err := c.store.CreateMessage(ctx, tx, data.Message{
			ID:           message.Info.ID,
			ChatID:       chatID,
			SenderID:     message.Info.Sender.User,
			Conversation: message.Message.GetConversation(),
			//CreatedAt:    message.Info.Timestamp,
			CreatedAt: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to create message from user in data store: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to execute data store transaction: %w", err)
	}

	err = c.whatsmeowClient.MarkRead(
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

	if state == StateSyncing {
		c.logger.Debug(
			"skipping handling of message while client is syncing",
			zap.String("message_id", message.Info.ID),
		)
		return nil
	}

	err = c.handleChat(chatID)
	if err != nil {
		return fmt.Errorf("failed to handle chat: %w", err)
	}

	return nil
}
