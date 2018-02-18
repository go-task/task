package logger

import (
	"fmt"
	"io"
)

type Logger struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Verbose bool
}

func (l *Logger) Outf(s string, args ...interface{}) {
	if len(args) == 0 {
		s, args = "%s", []interface{}{s}
	}
	fmt.Fprintf(l.Stdout, s+"\n", args...)
}

func (l *Logger) VerboseOutf(s string, args ...interface{}) {
	if l.Verbose {
		l.Outf(s, args...)
	}
}

func (l *Logger) Errf(s string, args ...interface{}) {
	if len(args) == 0 {
		s, args = "%s", []interface{}{s}
	}
	fmt.Fprintf(l.Stderr, s+"\n", args...)
}

func (l *Logger) VerboseErrf(s string, args ...interface{}) {
	if l.Verbose {
		l.Errf(s, args...)
	}
}
