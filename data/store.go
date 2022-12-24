package data

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("resource not found")

type Store interface {
	User(ctx context.Context, id string) (*User, error)
}
