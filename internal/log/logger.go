package log

import (
	"io"
	"log"
	"os"
)

// Singleton logger
var logger struct {
	stdout  io.Writer
	stderr  io.Writer
	verbose bool
	color   bool
}

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)
	logger.stdout = io.Discard
	logger.stderr = io.Discard
	logger.verbose = false
	logger.color = true
}

func SetColor(color bool) {
	logger.color = color
}

func SetStdout(w io.Writer) {
	logger.stdout = w
}

func SetStderr(w io.Writer) {
	logger.stderr = w
}

func SetVerbose(verbose bool) {
	logger.verbose = verbose
}

func Print(v ...any) {
	log.Print(v...)
}

// Outf prints stuff to STDOUT.
func Outf(color Color, s string, args ...any) {
	FOutf(logger.stdout, color, s, args...)
}

// FOutf prints stuff to the given writer.
func FOutf(w io.Writer, color Color, s string, args ...any) {
	if len(args) == 0 {
		s, args = "%s", []any{s}
	}
	if !logger.color {
		color = Default
	}
	print := color()
	print(w, s, args...)
}

// VerboseOutf prints stuff to STDOUT if verbose mode is enabled.
func VerboseOutf(color Color, s string, args ...any) {
	if logger.verbose {
		Outf(color, s, args...)
	}
}

// Errf prints stuff to STDERR.
func Errf(color Color, s string, args ...any) {
	if len(args) == 0 {
		s, args = "%s", []any{s}
	}
	if !logger.color {
		color = Default
	}
	print := color()
	print(logger.stderr, s, args...)
}

// VerboseErrf prints stuff to STDERR if verbose mode is enabled.
func VerboseErrf(color Color, s string, args ...any) {
	if logger.verbose {
		Errf(color, s, args...)
	}
}

func Fatal(v ...any) {
	Errf(Red, "%v\n", v...)
	os.Exit(1)
}
