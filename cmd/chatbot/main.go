package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const programName = "chatbot"

func main() {
	var statusCode int
	defer func() {
		os.Exit(statusCode)
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var logger *zap.Logger
	defer logger.Sync()

	cfg, err := newConfig(os.Args[1:])
	if err != nil {
		log.Printf("failed to construct config: %s", err)
		statusCode = 2
		return
	}

	logger, err = newLogger(cfg.development)
	if err != nil {
		log.Printf("failed to construct logger: %s", err)
		statusCode = 1
		return
	}

	err = run(ctx, logger, cfg)
	if err != nil {
		logger.Error("failed to run", zap.Error(err))
		statusCode = 1
		return
	}
}

func newLogger(development bool) (*zap.Logger, error) {
	var cfg zap.Config

	if development {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
	}

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}
	logger = logger.Named(programName)

	return logger, nil
}
