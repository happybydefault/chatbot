package memory

import (
	"sync"

	"github.com/happybydefault/chatbot/data"
)

type Store struct {
	mu    sync.RWMutex
	chats map[string]*data.Chat
}

func NewStore(chatIDs []string) *Store {
	chats := make(map[string]*data.Chat, len(chatIDs))
	for _, id := range chatIDs {
		chats[id] = &data.Chat{ID: id}
	}

	return &Store{
		chats: chats,
	}
}
