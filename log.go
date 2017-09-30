package task

import (
	"fmt"
)

func (e *Executor) outf(s string, args ...interface{}) {
	if len(args) == 0 {
		s, args = "%s", []interface{}{s}
	}
	fmt.Fprintf(e.Stdout, s+"\n", args...)
}

func (e *Executor) verboseOutf(s string, args ...interface{}) {
	if e.Verbose {
		e.outf(s, args...)
	}
}

func (e *Executor) errf(s string, args ...interface{}) {
	if len(args) == 0 {
		s, args = "%s", []interface{}{s}
	}
	fmt.Fprintf(e.Stderr, s+"\n", args...)
}

func (e *Executor) verboseErrf(s string, args ...interface{}) {
	if e.Verbose {
		e.errf(s, args...)
	}
}
