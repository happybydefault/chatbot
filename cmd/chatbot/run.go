package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	pgxstd "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"github.com/happybydefault/chatbot"
	"github.com/happybydefault/chatbot/postgres"
)

func run(ctx context.Context, logger *zap.Logger, cfg config) error {
	connConfig, err := pgx.ParseConfig(cfg.postgresConnString)
	if err != nil {
		return fmt.Errorf("failed to parse Postgres connection string: %w", err)
	}
	db := pgxstd.OpenDB(*connConfig)
	defer func() {
		err := db.Close()
		if err != nil {
			logger.Error("failed to close Postgres database connection", zap.Error(err))
		}
	}()

	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	store, err := postgres.NewStore(ctx, cfg.postgresConnString, logger.Named("postgres-store"))
	if err != nil {
		return fmt.Errorf("failed to construct Postgres store: %w", err)
	}
	defer store.Close()

	chatbotConfig := chatbot.Config{
		Logger:       logger,
		Store:        store,
		WhatsmeowDB:  db,
		OpenAIAPIKey: cfg.openAIAPIKey,
	}

	client, err := chatbot.NewClient(chatbotConfig)
	if err != nil {
		return fmt.Errorf("failed to construct chatbot client: %w", err)
	}
	defer func() {
		err := client.Stop()
		if err != nil {
			logger.Error("failed to stop chatbot client", zap.Error(err))
		}
	}()

	errChan := make(chan error, 1)
	go func() {
		err := client.Start()
		if err != nil {
			errChan <- fmt.Errorf("failed to start chatbot client: %w", err)
		}
		errChan <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
