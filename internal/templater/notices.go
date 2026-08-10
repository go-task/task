package templater

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
)

// NoticeFunc receives the deprecation notices that sprout emits when a
// template calls a deprecated function name or uses a deprecated argument
// order.
type NoticeFunc func(format string, args ...any)

// noticeSink is set once the Executor has built its logger. Until then — and
// the function map is built in an init(), long before that — notices are
// dropped rather than written to sprout's default stdout handler, which would
// corrupt the JSON and group output styles.
var noticeSink atomic.Pointer[NoticeFunc]

// SetNoticeSink installs the destination for template deprecation notices.
// Passing nil silences them again.
func SetNoticeSink(fn NoticeFunc) {
	noticeSeen.Clear()
	if fn == nil {
		noticeSink.Store(nil)
		return
	}
	noticeSink.Store(&fn)
}

type noticeHandler struct{}

func (noticeHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelWarn && noticeSink.Load() != nil
}

// noticeSeen keeps each distinct notice to a single line. Task renders the
// same templates several times per run — once per compilation pass — so
// without this a lone deprecated call would be reported over and over.
var noticeSeen sync.Map

func (noticeHandler) Handle(_ context.Context, record slog.Record) error {
	fn := noticeSink.Load()
	if fn == nil {
		return nil
	}
	if _, dup := noticeSeen.LoadOrStore(record.Message, struct{}{}); dup {
		return nil
	}
	(*fn)("task: %s\n", record.Message)
	return nil
}

func (h noticeHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h noticeHandler) WithGroup(string) slog.Handler { return h }
