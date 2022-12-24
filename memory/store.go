package memory

import "github.com/happybydefault/chatbot/data"

type Store struct {
	users map[string]*data.User
}

func NewStore(userIDs []string) *Store {
	users := make(map[string]*data.User, len(userIDs))
	for _, id := range userIDs {
		users[id] = &data.User{ID: id}
	}

	return &Store{
		users: users,
	}
}
