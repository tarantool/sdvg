package handlers

import (
	"context"
	"log/slog"
)

var DummyLogger = slog.New(dummyHandler{})

type dummyHandler struct{}

func (h dummyHandler) Enabled(context.Context, slog.Level) bool { return false }

func (h dummyHandler) Handle(context.Context, slog.Record) error { return nil }

func (h dummyHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h dummyHandler) WithGroup(string) slog.Handler {
	return h
}
