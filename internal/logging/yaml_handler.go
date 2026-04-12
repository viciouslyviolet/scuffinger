package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync"
	"time"
)

// YAMLHandler is a slog.Handler that writes log records in YAML format.
// Each record is a separate YAML document delimited by "---".
type YAMLHandler struct {
	opts  slog.HandlerOptions
	mu    *sync.Mutex
	w     io.Writer
	attrs []slog.Attr
	group string
}

// NewYAMLHandler creates a new YAMLHandler.
func NewYAMLHandler(w io.Writer, opts *slog.HandlerOptions) *YAMLHandler {
	h := &YAMLHandler{
		w:  w,
		mu: &sync.Mutex{},
	}
	if opts != nil {
		h.opts = *opts
	}
	return h
}

func (h *YAMLHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *YAMLHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// YAML document separator
	fmt.Fprintln(h.w, "---")
	fmt.Fprintf(h.w, "time: %q\n", r.Time.Format(time.RFC3339Nano))
	fmt.Fprintf(h.w, "level: %s\n", r.Level.String())
	fmt.Fprintf(h.w, "msg: %q\n", r.Message)

	// Pre-configured attrs (from With / WithAttrs)
	for _, a := range h.attrs {
		writeYAMLAttr(h.w, h.group, a)
	}

	// Per-record attrs
	r.Attrs(func(a slog.Attr) bool {
		writeYAMLAttr(h.w, h.group, a)
		return true
	})

	return nil
}

func (h *YAMLHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &YAMLHandler{
		opts:  h.opts,
		mu:    h.mu,
		w:     h.w,
		attrs: append(slices.Clone(h.attrs), attrs...),
		group: h.group,
	}
}

func (h *YAMLHandler) WithGroup(name string) slog.Handler {
	newGroup := name
	if h.group != "" {
		newGroup = h.group + "." + name
	}
	return &YAMLHandler{
		opts:  h.opts,
		mu:    h.mu,
		w:     h.w,
		attrs: slices.Clone(h.attrs),
		group: newGroup,
	}
}

// writeYAMLAttr writes a single slog.Attr as a YAML key/value line.
func writeYAMLAttr(w io.Writer, prefix string, a slog.Attr) {
	key := a.Key
	if prefix != "" {
		key = prefix + "." + key
	}

	switch a.Value.Kind() {
	case slog.KindGroup:
		for _, ga := range a.Value.Group() {
			writeYAMLAttr(w, key, ga)
		}
	case slog.KindString:
		fmt.Fprintf(w, "%s: %q\n", key, a.Value.String())
	case slog.KindInt64:
		fmt.Fprintf(w, "%s: %d\n", key, a.Value.Int64())
	case slog.KindFloat64:
		fmt.Fprintf(w, "%s: %.6f\n", key, a.Value.Float64())
	case slog.KindBool:
		fmt.Fprintf(w, "%s: %t\n", key, a.Value.Bool())
	case slog.KindTime:
		fmt.Fprintf(w, "%s: %q\n", key, a.Value.Time().Format(time.RFC3339Nano))
	case slog.KindDuration:
		fmt.Fprintf(w, "%s: %q\n", key, a.Value.Duration().String())
	default:
		fmt.Fprintf(w, "%s: %q\n", key, a.Value.String())
	}
}
