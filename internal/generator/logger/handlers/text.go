package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"
)

type TextHandler struct {
	slog.Handler
	l *log.Logger
}

func (h *TextHandler) Handle(_ context.Context, r slog.Record) error {
	timeStr := r.Time.Format("2006/01/02 15:04:05")
	levelStr := r.Level.String()
	msg := r.Message

	var attrs []string

	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value.Any()))

		return true
	})

	attrsStr := strings.Join(attrs, " ")

	h.l.Println(timeStr, levelStr, msg, attrsStr)

	return nil
}

func NewTextHandler(out io.Writer, options *slog.HandlerOptions) *TextHandler {
	return &TextHandler{
		Handler: slog.NewTextHandler(out, options),
		l:       log.New(out, "", 0),
	}
}
