package logger

import (
	"io"
	"os"
	"strconv"

	"github.com/fatih/color"
)

type Color func() PrintFunc
type PrintFunc func(io.Writer, string, ...interface{})

func Getenvcolor(envVar string, defaultColor color.Attribute) color.Attribute {
	override, err := strconv.Atoi(os.Getenv(envVar))
	if err != nil {
		return defaultColor
	}
	return color.Attribute(override)
}
func Default() PrintFunc { return color.New(Getenvcolor("TASK_COLOR_RESET", color.Reset)).FprintfFunc() }
func Blue() PrintFunc    { return color.New(Getenvcolor("TASK_COLOR_BLUE", color.FgBlue)).FprintfFunc() }
func Green() PrintFunc   { return color.New(Getenvcolor("TASK_COLOR_GREEN", color.FgGreen)).FprintfFunc() }
func Cyan() PrintFunc    { return color.New(Getenvcolor("TASK_COLOR_CYAN", color.FgCyan)).FprintfFunc() }
func Yellow() PrintFunc  { return color.New(Getenvcolor("TASK_COLOR_YELLOW", color.FgYellow)).FprintfFunc() }
func Magenta() PrintFunc { return color.New(Getenvcolor("TASK_COLOR_MAGENTA", color.FgMagenta)).FprintfFunc() }
func Red() PrintFunc     { return color.New(Getenvcolor("TASK_COLOR_RED", color.FgRed)).FprintfFunc() }

// Logger is just a wrapper that prints stuff to STDOUT or STDERR,
// with optional color.
type Logger struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Verbose bool
	Color   bool
}

// Outf prints stuff to STDOUT.
func (l *Logger) Outf(color Color, s string, args ...interface{}) {
	if len(args) == 0 {
		s, args = "%s", []interface{}{s}
	}
	if !l.Color {
		color = Default
	}
	print := color()
	print(l.Stdout, s+"\n", args...)
}

// VerboseOutf prints stuff to STDOUT if verbose mode is enabled.
func (l *Logger) VerboseOutf(color Color, s string, args ...interface{}) {
	if l.Verbose {
		l.Outf(color, s, args...)
	}
}

// Errf prints stuff to STDERR.
func (l *Logger) Errf(color Color, s string, args ...interface{}) {
	if len(args) == 0 {
		s, args = "%s", []interface{}{s}
	}
	if !l.Color {
		color = Default
	}
	print := color()
	print(l.Stderr, s+"\n", args...)
}

// VerboseErrf prints stuff to STDERR if verbose mode is enabled.
func (l *Logger) VerboseErrf(color Color, s string, args ...interface{}) {
	if l.Verbose {
		l.Errf(color, s, args...)
	}
}
