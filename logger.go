package chatbot

import (
	"context"

	"go.uber.org/zap"
)

type loggerContextKey struct{}

func contextWithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

func loggerFromContext(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(loggerContextKey{}).(*zap.Logger)
	if !ok {
		return zap.NewNop()
	}

	return logger
}
