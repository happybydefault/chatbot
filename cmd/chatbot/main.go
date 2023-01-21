package main

import (
	"context"
	"errors"
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

	cfg, err := newConfig(os.Args[1:])
	if err != nil {
		log.Printf("failed to construct config: %s", err)
		statusCode = 2
		return
	}

	logger, err := newLogger(cfg.development)
	if err != nil {
		log.Printf("failed to construct logger: %s", err)
		statusCode = 1
		return
	}
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-stopChan
		signal.Stop(stopChan)
		logger.Info("received stopping signal", zap.String("signal", sig.String()))
		cancel()
	}()

	err = run(ctx, logger, cfg)
	if err != nil && !errors.Is(err, context.Canceled) {
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
