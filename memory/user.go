package memory

import (
	"context"

	"github.com/happybydefault/chatbot/data"
)

func (s *Store) User(ctx context.Context, id string) (*data.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return nil, data.ErrNotFound
	}

	return user, nil
}
