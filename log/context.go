package log

import (
	"context"

	"go.uber.org/zap"
)

type logKey struct{}

var key logKey
var defaultLogger *zap.Logger

func init() {
	// TODO fix this up
	defaultLogger, _ = zap.NewProduction()
}

// FromContext returns a zap logger if one exists in the context,
// TODO : if not do we return nil or a default logger?? (default for now)
func FromContext(ctx context.Context) *zap.Logger {
	val, ok := ctx.Value(key).(*zap.Logger)
	if ok {
		return val
	}
	defaultLogger.Warn("logger not found in context, proceeding with defaultLogger")
	return defaultLogger
}

func NewContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, key, logger)
}
