package logger

import (
	"bufio"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/Ladicle/tabwriter"
	"github.com/fatih/color"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/term"
)

var (
	ErrPromptCancelled = errors.New("prompt cancelled")
	ErrNoTerminal      = errors.New("no terminal")
)

var (
	attrsReset       = envColor("COLOR_RESET", color.Reset)
	attrsFgBlue      = envColor("COLOR_BLUE", color.FgBlue)
	attrsFgGreen     = envColor("COLOR_GREEN", color.FgGreen)
	attrsFgCyan      = envColor("COLOR_CYAN", color.FgCyan)
	attrsFgYellow    = envColor("COLOR_YELLOW", color.FgYellow)
	attrsFgMagenta   = envColor("COLOR_MAGENTA", color.FgMagenta)
	attrsFgRed       = envColor("COLOR_RED", color.FgRed)
	attrsFgHiBlue    = envColor("COLOR_BRIGHT_BLUE", color.FgHiBlue)
	attrsFgHiGreen   = envColor("COLOR_BRIGHT_GREEN", color.FgHiGreen)
	attrsFgHiCyan    = envColor("COLOR_BRIGHT_CYAN", color.FgHiCyan)
	attrsFgHiYellow  = envColor("COLOR_BRIGHT_YELLOW", color.FgHiYellow)
	attrsFgHiMagenta = envColor("COLOR_BRIGHT_MAGENTA", color.FgHiMagenta)
	attrsFgHiRed     = envColor("COLOR_BRIGHT_RED", color.FgHiRed)
)

type (
	Color     func() PrintFunc
	PrintFunc func(io.Writer, string, ...any)
)

func Default() PrintFunc {
	return color.New(attrsReset...).FprintfFunc()
}

func Blue() PrintFunc {
	return color.New(attrsFgBlue...).FprintfFunc()
}

func Green() PrintFunc {
	return color.New(attrsFgGreen...).FprintfFunc()
}

func Cyan() PrintFunc {
	return color.New(attrsFgCyan...).FprintfFunc()
}

func Yellow() PrintFunc {
	return color.New(attrsFgYellow...).FprintfFunc()
}

func Magenta() PrintFunc {
	return color.New(attrsFgMagenta...).FprintfFunc()
}

func Red() PrintFunc {
	return color.New(attrsFgRed...).FprintfFunc()
}

func BrightBlue() PrintFunc {
	return color.New(attrsFgHiBlue...).FprintfFunc()
}

func BrightGreen() PrintFunc {
	return color.New(attrsFgHiGreen...).FprintfFunc()
}

func BrightCyan() PrintFunc {
	return color.New(attrsFgHiCyan...).FprintfFunc()
}

func BrightYellow() PrintFunc {
	return color.New(attrsFgHiYellow...).FprintfFunc()
}

func BrightMagenta() PrintFunc {
	return color.New(attrsFgHiMagenta...).FprintfFunc()
}

func BrightRed() PrintFunc {
	return color.New(attrsFgHiRed...).FprintfFunc()
}

func envColor(name string, defaultColor color.Attribute) []color.Attribute {
	// Fetch the environment variable
	override := env.GetTaskEnv(name)

	// First, try splitting the string by commas (RGB shortcut syntax) and if it
	// matches, then prepend the 256-color foreground escape sequence.
	// Otherwise, split by semicolons (ANSI color codes) and use them as is.
	attributeStrs := strings.Split(override, ",")
	if len(attributeStrs) == 3 {
		attributeStrs = slices.Concat([]string{"38", "2"}, attributeStrs)
	} else {
		attributeStrs = strings.Split(override, ";")
	}

	// Loop over the attributes and convert them to integers
	attributes := make([]color.Attribute, len(attributeStrs))
	for i, attributeStr := range attributeStrs {
		attribute, err := strconv.Atoi(attributeStr)
		if err != nil {
			return []color.Attribute{defaultColor}
		}
		attributes[i] = color.Attribute(attribute)
	}

	return attributes
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

func (l *Logger) Warnf(message string, args ...any) {
	l.Errf(Yellow, message, args...)
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

	l.Outf(color, "%s [%s/%s]: ", prompt, strings.ToLower(continueValues[0]), strings.ToUpper(defaultValue))

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
