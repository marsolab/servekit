package logkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/maxatome/go-testdeep/td"
)

func TestWithWriter(t *testing.T) {
	f := func(input io.Writer) {
		t.Helper()

		options := Options{writer: nil}
		WithWriter(input)(&options)

		td.Cmp(t, options.writer, input)
	}

	f(&bytes.Buffer{})
	f(nil)
}

func TestParseLevel(t *testing.T) {
	f := func(input string, expLevel slog.Level, expErr error) {
		t.Helper()

		got, err := ParseLevel(input)
		td.Cmp(t, err, expErr)
		td.Cmp(t, got, expLevel)
	}

	f("", slog.LevelInfo, ErrParseLevel)
	f("Nonexistent level", slog.LevelInfo, fmt.Errorf("%s %w", "Nonexistent level", ErrParseLevel))
	f("warn", slog.LevelWarn, nil)
	f("error", slog.LevelError, nil)
	f("info", slog.LevelInfo, nil)
	f("debug", slog.LevelDebug, nil)
	f("wArn", slog.LevelWarn, nil)
}

func TestNew_Defaults(t *testing.T) {
	// New without options should return a non-nil logger.
	logger := New()
	td.CmpNotNil(t, logger)
}

func TestNew_WithWriter(t *testing.T) {
	// Logger output should go to the provided writer.
	buf := &bytes.Buffer{}
	logger := New(WithWriter(buf))

	logger.Info("hello writer")

	td.Cmp(t, strings.Contains(buf.String(), "hello writer"), true)
}

func TestNew_WithJSON(t *testing.T) {
	// WithJSON should produce valid JSON log lines.
	buf := &bytes.Buffer{}
	logger := New(WithWriter(buf), WithJSON())

	logger.Info("json test", "key", "value")

	var parsed map[string]any
	err := json.Unmarshal(buf.Bytes(), &parsed)
	td.CmpNoError(t, err, "output should be valid JSON")
	td.Cmp(t, parsed["msg"], "json test")
	td.Cmp(t, parsed["key"], "value")
}

func TestNew_WithLevel(t *testing.T) {
	// WithLevel should filter messages below the configured level.
	buf := &bytes.Buffer{}
	logger := New(WithWriter(buf), WithJSON(), WithLevel(slog.LevelWarn))

	logger.Info("should not appear")
	td.Cmp(t, buf.String(), "")

	logger.Warn("should appear")
	td.Cmp(t, strings.Contains(buf.String(), "should appear"), true)
}

func TestNew_WithSource(t *testing.T) {
	// WithSource should include source information in the log output.
	buf := &bytes.Buffer{}
	logger := New(WithWriter(buf), WithJSON(), WithSource())

	logger.Info("source test")

	var parsed map[string]any
	err := json.Unmarshal(buf.Bytes(), &parsed)
	td.CmpNoError(t, err)
	td.CmpNotNil(t, parsed["source"], "JSON output should contain source field")
}

func TestNew_WithColor(t *testing.T) {
	// WithColor should produce output without error (uses tint handler).
	buf := &bytes.Buffer{}
	logger := New(WithWriter(buf), WithColor())

	logger.Info("color test")
	td.Cmp(t, strings.Contains(buf.String(), "color test"), true)
}

func TestNew_WithTimeFormat(t *testing.T) {
	// WithTimeFormat should apply the given time format to log output.
	buf := &bytes.Buffer{}
	format := time.RFC3339
	logger := New(WithWriter(buf), WithTimeFormat(format))

	logger.Info("time format test")

	output := buf.String()
	td.Cmp(t, len(output) > 0, true, "output should not be empty")
}

func TestNew_MultipleOptions(t *testing.T) {
	// Combining multiple options should work without error.
	buf := &bytes.Buffer{}
	logger := New(
		WithWriter(buf),
		WithLevel(slog.LevelDebug),
		WithJSON(),
		WithSource(),
	)

	logger.Debug("multi option test", "foo", "bar")

	var parsed map[string]any
	err := json.Unmarshal(buf.Bytes(), &parsed)
	td.CmpNoError(t, err)
	td.Cmp(t, parsed["msg"], "multi option test")
	td.Cmp(t, parsed["foo"], "bar")
	td.CmpNotNil(t, parsed["source"])
}

func TestNewNop(t *testing.T) {
	// NewNop should return a non-nil logger that produces no output.
	logger := NewNop()
	td.CmpNotNil(t, logger)

	// The nop logger handler should report Enabled as false.
	handler := logger.Handler()
	td.Cmp(t, handler.Enabled(context.Background(), slog.LevelError), false)
}

func TestNoopHandler_Enabled(t *testing.T) {
	// Enabled should always return false for all levels.
	h := &noopHandler{}
	levels := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}

	for _, lvl := range levels {
		td.Cmp(t, h.Enabled(context.Background(), lvl), false)
	}
}

func TestNoopHandler_Handle(t *testing.T) {
	// Handle should return nil without error.
	h := &noopHandler{}
	err := h.Handle(context.Background(), slog.Record{})
	td.CmpNoError(t, err)
}

func TestNoopHandler_WithAttrs(t *testing.T) {
	// WithAttrs should return the same handler.
	h := noopHandler{}
	result := h.WithAttrs([]slog.Attr{slog.String("key", "value")})
	td.Cmp(t, result, h)
}

func TestNoopHandler_WithGroup(t *testing.T) {
	// WithGroup should return the same handler.
	h := noopHandler{}
	result := h.WithGroup("group")
	td.Cmp(t, result, h)
}

func TestWithLevel_Option(t *testing.T) {
	// WithLevel should set the level on the Options LevelVar.
	o := Options{level: &slog.LevelVar{}}
	WithLevel(slog.LevelError)(&o)
	td.Cmp(t, o.level.Level(), slog.LevelError)
}

func TestWithJSON_Option(t *testing.T) {
	// WithJSON should set the withJSON flag to true.
	o := Options{}
	WithJSON()(&o)
	td.Cmp(t, o.withJSON, true)
}

func TestWithSource_Option(t *testing.T) {
	// WithSource should set the withSource flag to true.
	o := Options{}
	WithSource()(&o)
	td.Cmp(t, o.withSource, true)
}

func TestWithColor_Option(t *testing.T) {
	// WithColor should set the withColor flag to true.
	o := Options{}
	WithColor()(&o)
	td.Cmp(t, o.withColor, true)
}

func TestWithTimeFormat_Option(t *testing.T) {
	// WithTimeFormat should set the timeFormat field.
	o := Options{}
	WithTimeFormat(time.RFC3339)(&o)
	td.Cmp(t, o.timeFormat, time.RFC3339)
}

func TestError_Error(t *testing.T) {
	// Error type should return its string representation.
	e := Error("test error")
	td.Cmp(t, e.Error(), "test error")
}

func TestError_AsErrorInterface(t *testing.T) {
	// Error should satisfy the error interface.
	var err error = Error("some error")
	td.Cmp(t, err.Error(), "some error")
}

func TestErrParseLevel_Value(t *testing.T) {
	// ErrParseLevel constant should have the expected message.
	td.Cmp(t, ErrParseLevel.Error(), "string can't be parsed as Level, use: `error`, `warn`, `info`, `debug`")
}
