package chatbot

import (
	"database/sql"

	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

type Config struct {
	Logger       *zap.Logger
	Store        data.Store
	WhatsmeowDB  *sql.DB // Must be a Postgres database.
	OpenAIAPIKey string
}
