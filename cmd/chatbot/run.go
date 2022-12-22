package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/happybydefault/chatbot"
)

func run(ctx context.Context, logger *zap.Logger, cfg config) error {
	chatbotConfig := chatbot.Config{
		Logger:             logger,
		PostgresConnString: cfg.postgresConnString,
	}

	server, err := chatbot.NewServer(chatbotConfig)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	defer func() {
		err := server.Close()
		if err != nil {
			logger.Error("failed to close server", zap.Error(err))
		}
	}()

	err = server.Serve(ctx)
	if err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}
