package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/happybydefault/chatbot/data"
)

func (s *Store) Chat(ctx context.Context, tx data.Tx, whatsappID string) (data.Chat, error) {
	query := "SELECT chat_id FROM chats WHERE chat_id = $1 LIMIT 1"

	row := tx.QueryRow(ctx, query, whatsappID)

	var chat data.Chat
	err := row.Scan(&chat.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return data.Chat{}, data.ErrNotFound
		}
		return data.Chat{}, fmt.Errorf("failed to scan row: %w", err)
	}

	return chat, nil
}
