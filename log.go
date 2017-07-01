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
