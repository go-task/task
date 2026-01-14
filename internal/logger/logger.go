package logger

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"

	"github.com/Ladicle/tabwriter"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/internal/term"
)

const (
	LevelTaskVerbose    = slog.LevelDebug     // VerboseOutf -> stdout
	LevelTaskVerboseErr = slog.LevelDebug + 1 // VerboseErrf -> stderr
	LevelTaskInfo       = slog.LevelInfo      // Outf -> stdout
	LevelTaskInfoErr    = slog.LevelInfo + 1  // Errf -> stderr
	LevelTask           = slog.LevelInfo + 2  // Taskf -> (only json/text)
	LevelTaskWarning    = slog.LevelWarn      // Warnf(yellow) -> stderr
	LevelTaskError      = slog.LevelError     // Errorf(red) -> stderr
)

var levelNames = map[slog.Leveler]string{
	LevelTaskVerbose:    "VERBOSE",
	LevelTaskVerboseErr: "VERBOSE",
	LevelTaskInfo:       "INFO",
	LevelTaskInfoErr:    "INFO",
	LevelTask:           "TASK",
}

var (
	ErrPromptCancelled = errors.New("prompt cancelled")
	ErrNoTerminal      = errors.New("no terminal")
	colorKey           = contextColorKey("color")
)

type (
	contextColorKey string
)

type LoggerOptions struct {
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	Verbose    bool
	Color      bool
	AssumeYes  bool
	AssumeTerm bool // Used for testing
	LogFormat  string
}

// Logger is just a wrapper that prints stuff to STDOUT or STDERR,
// with optional color.
type Logger struct {
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	Verbose    bool
	Color      bool
	AssumeYes  bool
	AssumeTerm bool // Used for testing

	slogger *slog.Logger
}

func NewLogger(opts LoggerOptions) *Logger {
	logger := Logger{
		Stdin:      opts.Stdin,
		Stdout:     opts.Stdout,
		Stderr:     opts.Stderr,
		Verbose:    opts.Verbose,
		Color:      opts.Color,
		AssumeYes:  opts.AssumeYes,
		AssumeTerm: opts.AssumeTerm,
	}
	level := LevelTaskInfo
	if opts.Verbose {
		level = LevelTaskVerbose
	}
	handlerOps := TaskLogHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					label, exists := levelNames[a.Value.Any().(slog.Level)]
					if exists {
						a.Value = slog.StringValue(label)
					}
				}
				return a
			},
		},
		Logger: &logger,
	}

	switch lf := strings.ToLower(opts.LogFormat); lf {
	case "json":
		logger.slogger = slog.New(slog.NewJSONHandler(logger.Stdout, &handlerOps.HandlerOptions))
	case "text":
		logger.slogger = slog.New(slog.NewTextHandler(logger.Stdout, &handlerOps.HandlerOptions))
	default:
		logger.slogger = slog.New(NewTaskLogHandler(&handlerOps))
	}

	return &logger
}

func (l *Logger) IsStructured() bool {
	if l.slogger != nil {
		switch l.slogger.Handler().(type) {
		case *slog.JSONHandler:
			return true
		case *slog.TextHandler:
			return true
		}
	}
	return false
}

// Outf prints stuff to STDOUT.
func (l *Logger) Outf(color Color, s string, args ...any) {
	s = fmt.Sprintf(s, args...)
	l.slogger.Log(context.WithValue(context.Background(), colorKey, color), LevelTaskInfo, s)
}

// VerboseOutf prints stuff to STDOUT if verbose mode is enabled.
func (l *Logger) VerboseOutf(color Color, s string, args ...any) {
	s = fmt.Sprintf(s, args...)
	l.slogger.Log(context.WithValue(context.Background(), colorKey, color), LevelTaskVerbose, s)
}

// Errf prints stuff to STDERR.
func (l *Logger) Errf(color Color, s string, args ...any) {
	s = fmt.Sprintf(s, args...)
	l.slogger.Log(context.WithValue(context.Background(), colorKey, color), LevelTaskInfoErr, s)
}

// VerboseErrf prints stuff to STDERR if verbose mode is enabled.
func (l *Logger) VerboseErrf(color Color, s string, args ...any) {
	s = fmt.Sprintf(s, args...)
	l.slogger.Log(context.WithValue(context.Background(), colorKey, color), LevelTaskVerboseErr, s)
}

// Taskf prints to json/text logger only, args are key/value pairs.
func (l *Logger) Taskf(message string, args ...any) {
	var color Color = Cyan
	l.slogger.Log(context.WithValue(context.Background(), colorKey, color), LevelTask, message, args...)
}

func (l *Logger) Warnf(message string, args ...any) {
	var color Color = Yellow
	s := fmt.Sprintf(message, args...)
	l.slogger.Log(context.WithValue(context.Background(), colorKey, color), LevelTaskWarning, s)
}

func (l *Logger) Errorf(message string, args ...any) {
	var color Color = Red
	s := fmt.Sprintf(message, args...)
	l.slogger.Log(context.WithValue(context.Background(), colorKey, color), LevelTaskError, s)
}

// FOutf prints stuff to the given writer.
func (l *Logger) FOutf(w io.Writer, color Color, s string, args ...any) {
	if !l.Color {
		color = Default
	}
	print := color()
	print(w, s, args...)
}

func (l *Logger) OutfDirect(color Color, s string, args ...any) {
	l.FOutf(l.Stdout, color, s, args...)
}

func (l *Logger) ErrfDirect(color Color, s string, args ...any) {
	l.FOutf(l.Stderr, color, s, args...)
}

func (l *Logger) Prompt(color Color, prompt string, defaultValue string, continueValues ...string) error {
	if l.AssumeYes {
		l.OutfDirect(color, "%s [assuming yes]\n", prompt)
		return nil
	}

	if !l.AssumeTerm && !term.IsTerminal() {
		return ErrNoTerminal
	}

	if len(continueValues) == 0 {
		return errors.New("no continue values provided")
	}

	l.OutfDirect(color, "%s [%s/%s]: ", prompt, strings.ToLower(continueValues[0]), strings.ToUpper(defaultValue))

	reader := bufio.NewReader(l.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if !slices.Contains(continueValues, input) {
		return ErrPromptCancelled
	}

	return nil
}

func (l *Logger) PrintExperiments() error {
	w := tabwriter.NewWriter(l.Stdout, 0, 8, 0, ' ', 0)
	for _, x := range experiments.List() {
		if !x.Active() {
			continue
		}
		l.FOutf(w, Yellow, "* ")
		l.FOutf(w, Green, x.Name)
		l.FOutf(w, Default, ": \t%s\n", x.String())
	}
	return w.Flush()
}
