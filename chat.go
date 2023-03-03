package chatbot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gpt "github.com/sashabaranov/go-gpt3"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/happybydefault/chatbot/data"
)

type message struct {
	*events.Message
	clientState State
}

type Chat struct {
	client *Client
	logger *zap.Logger

	id           string
	messagesChan chan message
	wg           sync.WaitGroup

	mu              sync.Mutex
	pendingMessages atomic.Int32
}

func (c *Client) newChat(id string) *Chat {
	logger := c.logger.With(zap.String("chat_id", id))

	return &Chat{
		client:       c,
		logger:       logger,
		id:           id,
		messagesChan: make(chan message),
	}
}

func (c *Chat) close() {
	c.wg.Wait()
	close(c.messagesChan)
}

func (c *Chat) handleMessages() {
	for msg := range c.messagesChan {
		msg := msg

		c.wg.Add(1)
		go func() {
			defer c.wg.Done()

			logger := c.logger.With(zap.String("message_id", msg.Info.ID))

			err := c.handleMessage(msg)
			if err != nil {
				logger.Error("failed to handle chat message", zap.Error(err))
			}
		}()
	}
}

func (c *Chat) handleMessage(msg message) error {
	logger := c.logger.With(zap.String("message_id", msg.Info.ID))

	isAllowed, err := c.isAllowed()
	if err != nil {
		return fmt.Errorf("failed to check if chat is allowed: %w", err)
	}
	if !isAllowed {
		logger.Debug("skipped message because chat is not allowed")
		return nil
	}

	err = c.markAsRead(msg.Message)
	if err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	if msg.Message.Message.GetConversation() == "" {
		logger.Debug("ignored message with empty conversation")
		return nil
	}

	c.pendingMessages.Add(1)
	defer c.pendingMessages.Add(-1)

	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = c.storeMessageReceived(ctx, msg.Message)
	if err != nil {
		return fmt.Errorf("failed to store received message: %w", err)
	}

	if msg.clientState != StateSynced {
		logger.Debug("skipped responding to chat because client is not synced")
		return nil
	}
	if c.pendingMessages.Load() > 1 {
		logger.Debug("skipped responding to chat because there are pending messages")
		return nil
	}

	err = c.respond()
	if err != nil {
		return fmt.Errorf("failed to respond to chat: %w", err)
	}

	return nil
}

func (c *Chat) isAllowed() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.client.execTx(ctx, sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	}, func(tx data.Tx) error {
		_, err := c.client.store.Chat(ctx, tx, c.id)
		if err != nil {
			return fmt.Errorf("failed to get chat from data store: %w", err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to execute data store transaction: %w", err)
	}

	return true, nil
}

func (c *Chat) markAsRead(msg *events.Message) error {
	return c.client.whatsmeowClient.MarkRead(
		[]types.MessageID{
			msg.Info.ID,
		},
		time.Now(),
		msg.Info.Chat.ToNonAD(),
		msg.Info.Sender.ToNonAD(),
	)
}

func (c *Chat) storeMessageReceived(ctx context.Context, msg *events.Message) error {
	err := c.client.execTx(ctx, sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}, func(tx data.Tx) error {
		err := c.client.store.CreateMessage(ctx, tx, data.Message{
			ID:           msg.Info.ID,
			ChatID:       msg.Info.Chat.User,
			SenderID:     msg.Info.Sender.User,
			Conversation: msg.Message.GetConversation(),
			Timestamp:    msg.Info.Timestamp,
			CreatedAt:    time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to create message in data store: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to execute data store transaction: %w", err)
	}

	return nil
}

func (c *Chat) respond() error {
	c.logger.Info("responding chat", zap.String("chat_id", c.id))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var messages []data.Message
	err := c.client.execTx(ctx, sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	}, func(tx data.Tx) error {
		var err error

		messages, err = c.client.store.Messages(ctx, tx, c.id)
		if err != nil {
			return fmt.Errorf("failed to get messages from data store: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to execute data store transaction: %w", err)
	}

	if len(messages) == 0 {
		return errors.New("chat has no messages")
	}

	jid := types.NewJID(c.id, types.DefaultUserServer)

	err = c.client.whatsmeowClient.SendChatPresence(jid, types.ChatPresenceComposing, "")
	if err != nil {
		return fmt.Errorf("failed to send chat composing presence: %w", err)
	}

	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()

	conversations := make([]string, 0, len(messages))
	for _, msg := range messages {
		conversations = append(
			conversations,
			fmt.Sprintf(
				"%s:\n'''\n%s\n'''",
				msg.SenderID,
				msg.Conversation,
			),
		)
	}

	completionMessages := make([]gpt.ChatCompletionMessage, 0, len(messages)+1)

	// TODO: Maybe move this initial system message out of the function. Maybe use (Go) text templates.
	completionMessages = append(completionMessages, gpt.ChatCompletionMessage{
		Role: "system",
		Content: "The following is a conversation with an AI called Chatbot, the smartest of all beings." +
			" The assistant is helpful, creative, clever, and very friendly.",
	})

	for _, msg := range messages {
		var role string
		if msg.SenderID == c.client.whatsmeowClient.Store.ID.User {
			role = "assistant"
		} else {
			role = "user" // TODO: Support multiple users in order to fix group chats, if the completion API allows it.
		}

		completionMessages = append(completionMessages, gpt.ChatCompletionMessage{
			Role:    role,
			Content: msg.Conversation,
		})
	}

	completionResponse, err := c.client.completion(ctx, completionMessages)
	if err != nil {
		return fmt.Errorf("failed to get completion response: %w", err)
	}
	if len(completionResponse.Choices) == 0 {
		return fmt.Errorf("received empty slice of completion choices")
	}

	conversationResponse := completionResponse.Choices[0].Message
	if conversationResponse.Role != "assistant" {
		c.logger.Warn(
			"received completion response with unexpected role",
			zap.String("role", conversationResponse.Role),
		)
		return nil
	}
	conversationResponse.Content = strings.TrimSpace(conversationResponse.Content)

	response := &waProto.Message{
		Conversation: proto.String(conversationResponse.Content),
	}

	// Make sure there is a delay between receiving a message and sending a response,
	// to avoid being tagged as a bot and getting banned.
	<-timer.C

	report, err := c.client.whatsmeowClient.SendMessage(ctx, jid, "", response)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	c.logger.Debug("sent message", zap.String("sent_message_id", report.ID))

	err = c.client.execTx(ctx, sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}, func(tx data.Tx) error {
		err := c.client.store.CreateMessage(ctx, tx, data.Message{
			ID:           report.ID,
			ChatID:       c.id,
			SenderID:     c.client.whatsmeowClient.Store.ID.User,
			Conversation: conversationResponse.Content,
			Timestamp:    report.Timestamp,
			CreatedAt:    time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to create message from chatbot in data store: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to execute data store transaction: %w", err)
	}

	return nil
}
