package data

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("resource not found")

type Store interface {
	User(ctx context.Context, id string) (*User, error)

	Chat(ctx context.Context, id string) (*Chat, error)
	AppendMessage(ctx context.Context, chatID string, message Message) error
}
