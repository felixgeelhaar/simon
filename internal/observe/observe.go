package observe

import (
	"context"
	"io"

	"github.com/felixgeelhaar/bolt/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("simon")

// Observer handles logging and tracing
type Observer struct {
	log *bolt.Logger
}

func New(out io.Writer) *Observer {
	// Initialize bolt logger with a console handler for demo
	handler := bolt.NewConsoleHandler(out)
	l := bolt.New(handler)

	return &Observer{
		log: l,
	}
}

func NewJSON(out io.Writer) *Observer {
	handler := bolt.NewJSONHandler(out)
	l := bolt.New(handler)

	return &Observer{
		log: l,
	}
}

// Log returns the underlying logger
func (o *Observer) Log() *bolt.Logger {
	return o.log
}

// StartSpan starts a new OTel span
func (o *Observer) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return tracer.Start(ctx, name)
}

// Close ensures any buffered logs or traces are flushed (placeholder)
func (o *Observer) Close() error {
	return nil
}
