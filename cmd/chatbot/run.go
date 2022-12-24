package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/happybydefault/chatbot"
	"github.com/happybydefault/chatbot/memory"
)

func run(ctx context.Context, logger *zap.Logger, cfg config) error {
	store := memory.NewStore(cfg.userIDs)

	chatbotConfig := chatbot.Config{
		Logger:             logger,
		Store:              store,
		PostgresConnString: cfg.postgresConnString,
	}

	server, err := chatbot.NewServer(chatbotConfig)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	err = server.Serve(ctx)
	if err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}
