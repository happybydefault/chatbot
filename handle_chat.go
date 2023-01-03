package chatbot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/happybydefault/chatbot/data"
)

func (c *Client) handleChat(chatID string) error {
	c.logger.Info("handling chat", zap.String("chat_id", chatID))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var (
		chat     data.Chat
		messages []data.Message
	)
	err := c.execTx(ctx, sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	}, func(tx data.Tx) error {
		var err error

		chat, err = c.store.Chat(ctx, tx, chatID)
		if err != nil {
			return fmt.Errorf("failed to get chat from data store: %w", err)
		}

		messages, err = c.store.Messages(ctx, tx, chatID)
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

	jid := types.NewJID(chatID, types.DefaultUserServer)

	err = c.whatsmeowClient.SendChatPresence(jid, types.ChatPresenceComposing, "")
	if err != nil {
		return fmt.Errorf("failed to send chat composing presence: %w", err)
	}

	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()

	// TODO: Move prefix out of the function. Maybe use (Go) text templates.
	// TODO: Maybe encode conversation between the person and the AI as JSON.
	prefix := fmt.Sprintf(
		"The following is a conversation with an AI called Chatbot, the smartest of all beings."+
			" The assistant is helpful, creative, clever, and very friendly. The Chatbot's ID is %q.",
		c.whatsmeowClient.Store.ID.User,
	)

	conversations := make([]string, 0, len(messages))
	for _, message := range messages {
		conversations = append(
			conversations,
			fmt.Sprintf(
				"%s:\n'''\n%s\n'''",
				message.SenderID,
				message.Conversation,
			),
		)
	}

	prompt := fmt.Sprintf(
		"%s\n\n%s\n\n%s:\n'''",
		prefix,
		strings.Join(conversations, "\n\n"),
		c.whatsmeowClient.Store.ID.User,
	)

	completionResponse, err := c.completion(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to get completion response: %w", err)
	}
	if len(completionResponse.Choices) == 0 {
		return fmt.Errorf("received empty slice of completion choices")
	}

	conversationResponse := completionResponse.Choices[0].Text
	conversationResponse = strings.TrimSpace(conversationResponse)

	response := &waProto.Message{
		Conversation: proto.String(conversationResponse),
	}

	// Make sure there is a delay between receiving a message and sending a response,
	// to avoid being tagged as a bot and getting banned.
	<-timer.C

	report, err := c.whatsmeowClient.SendMessage(ctx, jid, "", response)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	c.logger.Debug("sent message", zap.String("sent_message_id", report.ID))

	err = c.execTx(ctx, sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}, func(tx data.Tx) error {
		err := c.store.CreateMessage(ctx, tx, data.Message{
			ID:           report.ID,
			Conversation: conversationResponse,
			ChatID:       chat.ID,
			SenderID:     c.whatsmeowClient.Store.ID.User,
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
