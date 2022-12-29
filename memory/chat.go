package memory

import (
	"context"
	"database/sql"

	"github.com/happybydefault/chatbot/data"
)

func (s *Store) Chat(ctx context.Context, tx *sql.Tx, id string) (*data.Chat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chat, ok := s.chats[id]
	if !ok {
		return nil, data.ErrNotFound
	}

	return chat, nil
}

func (s *Store) AppendMessage(ctx context.Context, chatID string, message data.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	chat, ok := s.chats[chatID]
	if !ok {
		return data.ErrNotFound
	}

	chat.Messages = append(chat.Messages, message)

	return nil
}
