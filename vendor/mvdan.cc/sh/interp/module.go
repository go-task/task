// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"context"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// Ctxt is the type passed to all the module functions. It contains some
// of the current state of the Runner, as well as some fields necessary
// to implement some of the modules.
type Ctxt struct {
	Context context.Context
	Env     []string
	Dir     string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// ModuleExec is the module responsible for executing a program. It is
// executed for all CallExpr nodes where the name is neither a declared
// function nor a builtin.
//
// Use a return error of type ExitCode to set the exit code. A nil error
// has the same effect as ExitCode(0). If the error is of any other
// type, the interpreter will come to a stop.
//
// TODO: replace name with path, to avoid the common "path :=
// exec.LookPath(name)"?
type ModuleExec func(ctx Ctxt, name string, args []string) error

func DefaultExec(ctx Ctxt, name string, args []string) error {
	cmd := exec.CommandContext(ctx.Context, name, args...)
	cmd.Env = ctx.Env
	cmd.Dir = ctx.Dir
	cmd.Stdin = ctx.Stdin
	cmd.Stdout = ctx.Stdout
	cmd.Stderr = ctx.Stderr
	err := cmd.Run()
	switch x := err.(type) {
	case *exec.ExitError:
		// started, but errored - default to 1 if OS
		// doesn't have exit statuses
		if status, ok := x.Sys().(syscall.WaitStatus); ok {
			return ExitCode(status.ExitStatus())
		}
		return ExitCode(1)
	case *exec.Error:
		// did not start
		// TODO: can this be anything other than
		// "command not found"?
		return ExitCode(127)
		// TODO: print something?
	default:
		return nil
	}
}

// ModuleOpen is the module responsible for opening a file. It is
// executed for all files that are opened directly by the shell, such as
// in redirects. Files opened by executed programs are not included.
//
// The path parameter is absolute and has been cleaned.
//
// Use a return error of type *os.PathError to have the error printed to
// stderr and the exit code set to 1. If the error is of any other type,
// the interpreter will come to a stop.
//
// TODO: What about stat calls? They are used heavily in the builtin
// test expressions, and also when doing a cd. Should they have a
// separate module?
type ModuleOpen func(ctx Ctxt, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error)

func DefaultOpen(ctx Ctxt, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	return os.OpenFile(path, flag, perm)
}

func OpenDevImpls(next ModuleOpen) ModuleOpen {
	return func(ctx Ctxt, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		switch path {
		case "/dev/null":
			return devNull{}, nil
		}
		return next(ctx, path, flag, perm)
	}
}

var _ io.ReadWriteCloser = devNull{}

type devNull struct{}

func (devNull) Read(p []byte) (int, error)  { return 0, io.EOF }
func (devNull) Write(p []byte) (int, error) { return len(p), nil }
func (devNull) Close() error                { return nil }
