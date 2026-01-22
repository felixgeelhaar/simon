package observe

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	buf := &bytes.Buffer{}
	obs := New(buf, true)

	if obs == nil {
		t.Fatal("expected non-nil Observer")
	}
	if obs.log == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	obs := NewJSON(buf, true)

	if obs == nil {
		t.Fatal("expected non-nil Observer")
	}
	if obs.log == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestObserver_Log(t *testing.T) {
	buf := &bytes.Buffer{}
	obs := New(buf, true)

	logger := obs.Log()
	if logger == nil {
		t.Fatal("expected non-nil logger from Log()")
	}

	// Log a message and verify it appears in the buffer
	logger.Info().Msg("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got %q", output)
	}
}

func TestObserver_StartSpan(t *testing.T) {
	buf := &bytes.Buffer{}
	obs := New(buf, true)

	ctx := context.Background()
	spanCtx, span := obs.StartSpan(ctx, "test-span")

	if spanCtx == nil {
		t.Fatal("expected non-nil context from StartSpan")
	}
	if span == nil {
		t.Fatal("expected non-nil span from StartSpan")
	}

	// End the span (cleanup)
	span.End()
}

func TestObserver_Close(t *testing.T) {
	buf := &bytes.Buffer{}
	obs := New(buf, true)

	err := obs.Close()
	if err != nil {
		t.Errorf("expected nil error from Close, got %v", err)
	}
}

func TestObserver_LogLevels(t *testing.T) {
	testCases := []struct {
		name  string
		level string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			obs := New(buf, true)
			logger := obs.Log()

			switch tc.level {
			case "debug":
				logger.Debug().Msg("test")
			case "info":
				logger.Info().Msg("test")
			case "warn":
				logger.Warn().Msg("test")
			case "error":
				logger.Error().Msg("test")
			}

			// Verify something was logged
			output := buf.String()
			if !strings.Contains(output, "test") {
				t.Errorf("expected output to contain 'test', got %q", output)
			}
		})
	}
}

func TestObserver_LogWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	obs := New(buf, true)

	obs.Log().Info().
		Str("session", "sess-123").
		Int("iteration", 5).
		Msg("iteration complete")

	output := buf.String()
	if !strings.Contains(output, "iteration complete") {
		t.Errorf("expected output to contain 'iteration complete', got %q", output)
	}
}
