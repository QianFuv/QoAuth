package oidc

import (
	"context"
	"io"
	"log/slog"
)

// SafeLogger records protocol event severity without arbitrary messages or attributes.
// The provider uses global slog; the command installs this before serving requests.
func SafeLogger(output io.Writer) *slog.Logger {
	return slog.New(safeHandler{next: slog.NewJSONHandler(output, nil)})
}

type safeHandler struct{ next slog.Handler }

func (h safeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}
func (h safeHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.next.Handle(ctx, slog.NewRecord(record.Time, record.Level, "protocol event", 0))
}
func (h safeHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h safeHandler) WithGroup(string) slog.Handler      { return h }
