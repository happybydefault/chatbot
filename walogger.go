package chatbot

import (
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"
)

// waLogger is a wrapper around zap.SugaredLogger to implement the waLog.Logger interface.
type waLogger struct {
	zap *zap.SugaredLogger
}

func newWALogger(zap *zap.Logger) *waLogger {
	return &waLogger{zap: zap.Sugar()}
}

func (l *waLogger) Warnf(msg string, args ...interface{}) {
	l.zap.Warnf(msg, args...)
}

func (l *waLogger) Errorf(msg string, args ...interface{}) {
	l.zap.Errorf(msg, args...)
}

func (l *waLogger) Infof(msg string, args ...interface{}) {
	l.zap.Infof(msg, args...)
}

func (l *waLogger) Debugf(msg string, args ...interface{}) {
	l.zap.Debugf(msg, args...)
}

func (l *waLogger) Sub(module string) waLog.Logger {
	return &waLogger{
		zap: l.zap.Named(module),
	}
}
