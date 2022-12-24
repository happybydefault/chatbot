package chatbot

import (
	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

type Config struct {
	Logger             *zap.Logger
	Store              data.Store
	PostgresConnString string
}
