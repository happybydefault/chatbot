package data

import "time"

type Message struct {
	ID           string
	ChatID       string
	SenderID     string
	Conversation string
	CreatedAt    time.Time
}
