package chatbot

import "go.uber.org/zap"

type Config struct {
	Logger             *zap.Logger
	PostgresConnString string
}
