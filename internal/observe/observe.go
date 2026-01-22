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

// New creates a new Observer with console output.
// If verbose is false, only warnings and errors are shown.
func New(out io.Writer, verbose bool) *Observer {
	handler := bolt.NewConsoleHandler(out)
	l := bolt.New(handler)

	if !verbose {
		l.SetLevel(bolt.WARN)
	}

	return &Observer{
		log: l,
	}
}

// NewJSON creates a new Observer with JSON output.
// If verbose is false, only warnings and errors are shown.
func NewJSON(out io.Writer, verbose bool) *Observer {
	handler := bolt.NewJSONHandler(out)
	l := bolt.New(handler)

	if !verbose {
		l.SetLevel(bolt.WARN)
	}

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
