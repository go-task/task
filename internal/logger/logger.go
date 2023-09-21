package logger

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/exp/slices"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/term"
)

var (
	ErrPromptCancelled = errors.New("prompt cancelled")
	ErrNoTerminal      = errors.New("no terminal")
)

type (
	Color     func() PrintFunc
	PrintFunc func(io.Writer, string, ...any)
)

func Default() PrintFunc {
	return color.New(envColor("TASK_COLOR_RESET", color.Reset)).FprintfFunc()
}

func Blue() PrintFunc {
	return color.New(envColor("TASK_COLOR_BLUE", color.FgBlue)).FprintfFunc()
}

func Green() PrintFunc {
	return color.New(envColor("TASK_COLOR_GREEN", color.FgGreen)).FprintfFunc()
}

func Cyan() PrintFunc {
	return color.New(envColor("TASK_COLOR_CYAN", color.FgCyan)).FprintfFunc()
}

func Yellow() PrintFunc {
	return color.New(envColor("TASK_COLOR_YELLOW", color.FgYellow)).FprintfFunc()
}

func Magenta() PrintFunc {
	return color.New(envColor("TASK_COLOR_MAGENTA", color.FgMagenta)).FprintfFunc()
}

func Red() PrintFunc {
	return color.New(envColor("TASK_COLOR_RED", color.FgRed)).FprintfFunc()
}

func envColor(env string, defaultColor color.Attribute) color.Attribute {
	if os.Getenv("FORCE_COLOR") != "" {
		color.NoColor = false
	}

	override, err := strconv.Atoi(os.Getenv(env))
	if err == nil {
		return color.Attribute(override)
	}
	return defaultColor
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
}

// Outf prints stuff to STDOUT.
func (l *Logger) Outf(color Color, s string, args ...any) {
	l.FOutf(l.Stdout, color, s, args...)
}

// FOutf prints stuff to the given writer.
func (l *Logger) FOutf(w io.Writer, color Color, s string, args ...any) {
	if len(args) == 0 {
		s, args = "%s", []any{s}
	}
	if !l.Color {
		color = Default
	}
	print := color()
	print(w, s, args...)
}

// VerboseOutf prints stuff to STDOUT if verbose mode is enabled.
func (l *Logger) VerboseOutf(color Color, s string, args ...any) {
	if l.Verbose {
		l.Outf(color, s, args...)
	}
}

// Errf prints stuff to STDERR.
func (l *Logger) Errf(color Color, s string, args ...any) {
	if len(args) == 0 {
		s, args = "%s", []any{s}
	}
	if !l.Color {
		color = Default
	}
	print := color()
	print(l.Stderr, s, args...)
}

// VerboseErrf prints stuff to STDERR if verbose mode is enabled.
func (l *Logger) VerboseErrf(color Color, s string, args ...any) {
	if l.Verbose {
		l.Errf(color, s, args...)
	}
}

func (l *Logger) Prompt(color Color, prompt string, defaultValue string, continueValues ...string) error {
	if l.AssumeYes {
		l.Outf(color, "%s [assuming yes]\n", prompt)
		return nil
	}

	if !l.AssumeTerm && !term.IsTerminal() {
		return ErrNoTerminal
	}

	if len(continueValues) == 0 {
		return errors.New("no continue values provided")
	}

	l.Outf(color, "%s [%s/%s]\n", prompt, strings.ToLower(continueValues[0]), strings.ToUpper(defaultValue))

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
