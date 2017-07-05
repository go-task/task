package task

import (
	"fmt"
)

func (e *Executor) println(args ...interface{}) {
	fmt.Fprintln(e.Stdout, args...)
}

func (e *Executor) printfln(format string, args ...interface{}) {
	fmt.Fprintf(e.Stdout, format+"\n", args...)
}

func (e *Executor) verbosePrintln(args ...interface{}) {
	if e.Verbose {
		e.println(args...)
	}
}

func (e *Executor) verbosePrintfln(format string, args ...interface{}) {
	if e.Verbose {
		e.printfln(format, args...)
	}
}
