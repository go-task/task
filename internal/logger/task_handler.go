package logger

import (
	"context"
	"log/slog"
	"os"
)

type TaskLogHandler struct {
	level  slog.Leveler
	logger *Logger
}

type TaskLogHandlerOptions struct {
	slog.HandlerOptions
	Logger *Logger
}

func NewTaskLogHandler(opts *TaskLogHandlerOptions) *TaskLogHandler {
	h := &TaskLogHandler{}
	if opts != nil {
		h.level = opts.Level
		h.logger = opts.Logger
	}
	if h.level == nil {
		h.level = LevelTaskInfo
	}
	if h.logger == nil {
		// Should be an impossible condition.
		h.errf(Red, "Task log handler not configured: no Logger object!")
		os.Exit(1)
	}
	return h
}

func (h *TaskLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *TaskLogHandler) Handle(ctx context.Context, r slog.Record) error {
	var color Color
	color, ok := ctx.Value(colorKey).(Color)
	if !ok {
		color = Default
	}
	switch {
	case r.Level == LevelTask:
		// NOP, only for json/text logging.
	case r.Level <= LevelTaskVerbose:
		h.outf(color, r.Message)
	case r.Level <= LevelTaskVerboseErr:
		h.errf(color, r.Message)
	case r.Level <= LevelTaskInfo:
		h.outf(color, r.Message)
	case r.Level <= LevelTaskInfoErr:
		h.errf(color, r.Message)
	case r.Level <= LevelTaskWarning:
		h.errf(color, r.Message)
	case r.Level <= LevelTaskError:
		h.errf(color, r.Message)
	default:
		h.errf(color, r.Message)
	}
	return nil
}

func (h *TaskLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return nil
}

func (h *TaskLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *TaskLogHandler) errf(color Color, s string, args ...any) {
	if !h.logger.Color {
		color = Default
	}
	print := color()
	print(h.logger.Stderr, s)
}

func (h *TaskLogHandler) outf(color Color, s string, args ...any) {
	if !h.logger.Color {
		color = Default
	}
	print := color()
	print(h.logger.Stdout, s)
}
