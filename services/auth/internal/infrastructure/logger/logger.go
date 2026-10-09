package logger

import (
	"context"
	"log/slog"
)

type Logger struct {
	logger *slog.Logger
}

func NewLogger(logger *slog.Logger) *Logger {
	return &Logger{
		logger: logger,
	}
}

func (l *Logger) Warn(ctx context.Context, msg string, keysAndValues ...any) {
	l.logger.WarnContext(ctx, msg, keysAndValues...)
}

func (l *Logger) Error(ctx context.Context, msg string, keysAndValues ...any) {
	l.logger.ErrorContext(ctx, msg, keysAndValues...)
}
