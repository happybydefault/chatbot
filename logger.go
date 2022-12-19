package chatbot

import (
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"
)

type Logger struct {
	zap *zap.SugaredLogger
}

func NewLogger(zap *zap.Logger) *Logger {
	return &Logger{zap: zap.Sugar()}
}

func (l *Logger) Warnf(msg string, args ...interface{}) {
	l.zap.Warnf(msg, args...)
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.zap.Errorf(msg, args...)
}

func (l *Logger) Infof(msg string, args ...interface{}) {
	l.zap.Infof(msg, args...)
}

func (l *Logger) Debugf(msg string, args ...interface{}) {
	l.zap.Debugf(msg, args...)
}

func (l *Logger) Sub(module string) waLog.Logger {
	return &Logger{
		zap: l.zap.Named(module),
	}
}
