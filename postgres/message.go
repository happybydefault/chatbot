package postgres

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

func (s *Store) Messages(ctx context.Context, tx data.Tx, chatID string) ([]data.Message, error) {
	query := "SELECT chat_id, sender_id, message_id, conversation FROM messages WHERE chat_id = $1"

	rows, err := tx.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			s.logger.Error("failed to close rows", zap.Error(err))
		}
	}()

	var messages []data.Message
	for rows.Next() {
		var message data.Message
		err := rows.Scan(&message.ChatID, &message.SenderID, &message.ID, &message.Conversation)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		messages = append(messages, message)
	}

	return messages, nil
}

func (s *Store) CreateMessage(ctx context.Context, tx data.Tx, message data.Message) error {
	query := "INSERT INTO messages (chat_id, sender_id, message_id, conversation) VALUES ($1, $2, $3, $4)"

	_, err := tx.Exec(ctx, query, message.ChatID, message.SenderID, message.ID, message.Conversation)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}
