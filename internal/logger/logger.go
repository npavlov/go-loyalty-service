package logger

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

type Logger struct {
	mx sync.Mutex
	lg zerolog.Logger
}

func NewLogger() *Logger {
	//nolint:exhaustruct
	var output io.Writer = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	return &Logger{
		mx: sync.Mutex{},
		lg: zerolog.New(output).With().Timestamp().Logger(),
	}
}

func (l *Logger) Get() *zerolog.Logger {
	return &l.lg
}

// SetLogLevel sets the global log level for all loggers.
func (l *Logger) SetLogLevel(level zerolog.Level) *Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(level)

	return l
}

func GetWithTrace(ctx context.Context, logger *zerolog.Logger) *zerolog.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		// If no valid span exists, return the logger as-is
		return logger
	}

	spanContext := span.SpanContext()

	updated := logger.With().
		Str("traceID", spanContext.TraceID().String()).
		Str("spanID", spanContext.SpanID().String()).Logger()

	return &updated
}
