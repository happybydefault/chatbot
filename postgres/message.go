package postgres

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

func (s *Store) CreateMessage(ctx context.Context, tx data.Tx, message data.Message) error {
	query := `INSERT INTO messages (chat_id, sender_id, message_id, conversation, "timestamp", created_at)
			  VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := tx.Exec(
		ctx,
		query,
		message.ChatID,
		message.SenderID,
		message.ID,
		message.Conversation,
		message.Timestamp,
		message.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (s *Store) Messages(ctx context.Context, tx data.Tx, chatID string) ([]data.Message, error) {
	query := `SELECT chat_id, sender_id, message_id, conversation, "timestamp", created_at
			  FROM messages
			  WHERE chat_id = $1
			  ORDER BY "timestamp"`

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
		message, err := s.scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		messages = append(messages, message)
	}

	return messages, nil
}

func (s *Store) AllMessagesSince(ctx context.Context, tx data.Tx, t time.Time) ([]data.Message, error) {
	query := `SELECT chat_id, sender_id, message_id, conversation, "timestamp", created_at
			  FROM messages
			  WHERE created_at >= $1
			  ORDER BY "timestamp"`

	rows, err := tx.Query(ctx, query, t)
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
		message, err := s.scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		messages = append(messages, message)
	}

	return messages, nil
}

func (s *Store) scanMessage(row data.Row) (data.Message, error) {
	var message data.Message
	err := row.Scan(
		&message.ChatID,
		&message.SenderID,
		&message.ID,
		&message.Conversation,
		&message.Timestamp,
		&message.CreatedAt,
	)
	if err != nil {
		return data.Message{}, fmt.Errorf("failed to scan row: %w", err)
	}

	return message, nil
}
