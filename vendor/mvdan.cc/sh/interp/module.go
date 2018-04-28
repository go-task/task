// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Ctxt is the type passed to all the module functions. It contains some
// of the current state of the Runner, as well as some fields necessary
// to implement some of the modules.
type Ctxt struct {
	Context     context.Context
	Env         Environ
	Dir         string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	KillTimeout time.Duration
}

// UnixPath fixes absolute unix paths on Windows, for example converting
// "C:\\CurDir\\dev\\null" to "/dev/null".
func (c *Ctxt) UnixPath(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}
	path = strings.TrimPrefix(path, c.Dir)
	return strings.Replace(path, `\`, `/`, -1)
}

// ModuleExec is the module responsible for executing a program. It is
// executed for all CallExpr nodes where the first argument is neither a
// declared function nor a builtin.
//
// Note that the name is included as the first argument. If path is an
// empty string, it means that the executable did not exist or was not
// found in $PATH.
//
// Use a return error of type ExitCode to set the exit code. A nil error
// has the same effect as ExitCode(0). If the error is of any other
// type, the interpreter will come to a stop.
type ModuleExec func(ctx Ctxt, path string, args []string) error

func DefaultExec(ctx Ctxt, path string, args []string) error {
	if path == "" {
		fmt.Fprintf(ctx.Stderr, "%q: executable file not found in $PATH\n", args[0])
		return ExitCode(127)
	}
	cmd := exec.Cmd{
		Path:   path,
		Args:   args,
		Env:    execEnv(ctx.Env),
		Dir:    ctx.Dir,
		Stdin:  ctx.Stdin,
		Stdout: ctx.Stdout,
		Stderr: ctx.Stderr,
	}

	err := cmd.Start()
	if err == nil {
		if done := ctx.Context.Done(); done != nil {
			go func() {
				<-done

				if ctx.KillTimeout <= 0 || runtime.GOOS == "windows" {
					_ = cmd.Process.Signal(os.Kill)
					return
				}

				// TODO: don't temporarily leak this goroutine
				// if the program stops itself with the
				// interrupt.
				go func() {
					time.Sleep(ctx.KillTimeout)
					_ = cmd.Process.Signal(os.Kill)
				}()
				_ = cmd.Process.Signal(os.Interrupt)
			}()
		}

		err = cmd.Wait()
	}

	switch x := err.(type) {
	case *exec.ExitError:
		// started, but errored - default to 1 if OS
		// doesn't have exit statuses
		if status, ok := x.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() && ctx.Context.Err() != nil {
				return ctx.Context.Err()
			}
			return ExitCode(status.ExitStatus())
		}
		return ExitCode(1)
	case *exec.Error:
		// did not start
		fmt.Fprintf(ctx.Stderr, "%v\n", err)
		return ExitCode(127)
	default:
		return err
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
		switch ctx.UnixPath(path) {
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
