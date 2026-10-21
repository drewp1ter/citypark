// Package logging builds the application logger and the HTTP middlewares that
// log requests and recover from panics.
package logging

import (
	"io"
	"log/slog"
)

// New returns a logger writing to w.
//
//	env == "production": JSON handler
//	otherwise:           text handler
//
// The level is Debug when debug is true, else Info.
func New(w io.Writer, env string, debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}
	if env == "production" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}
