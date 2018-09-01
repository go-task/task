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

// FromModuleContext returns the ModuleCtx value stored in ctx, if any.
func FromModuleContext(ctx context.Context) (ModuleCtx, bool) {
	mc, ok := ctx.Value(moduleCtxKey{}).(ModuleCtx)
	return mc, ok
}

type moduleCtxKey struct{}

// ModuleCtx is the data passed to all the module functions via a context value.
// It contains some of the current state of the Runner, as well as some fields
// necessary to implement some of the modules.
type ModuleCtx struct {
	Env         Environ
	Dir         string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	KillTimeout time.Duration
}

// UnixPath fixes absolute unix paths on Windows, for example converting
// "C:\\CurDir\\dev\\null" to "/dev/null".
func (mc ModuleCtx) UnixPath(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}
	path = strings.TrimPrefix(path, mc.Dir)
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
// Use a return error of type ExitStatus to set the exit status. A nil error has
// the same effect as ExitStatus(0). If the error is of any other type, the
// interpreter will come to a stop.
type ModuleExec func(ctx context.Context, path string, args []string) error

func (ModuleExec) isModule() {}

var DefaultExec = ModuleExec(func(ctx context.Context, path string, args []string) error {
	mc, _ := FromModuleContext(ctx)
	if path == "" {
		fmt.Fprintf(mc.Stderr, "%q: executable file not found in $PATH\n", args[0])
		return ExitStatus(127)
	}
	cmd := exec.Cmd{
		Path:   path,
		Args:   args,
		Env:    execEnv(mc.Env),
		Dir:    mc.Dir,
		Stdin:  mc.Stdin,
		Stdout: mc.Stdout,
		Stderr: mc.Stderr,
	}

	err := cmd.Start()
	if err == nil {
		if done := ctx.Done(); done != nil {
			go func() {
				<-done

				if mc.KillTimeout <= 0 || runtime.GOOS == "windows" {
					_ = cmd.Process.Signal(os.Kill)
					return
				}

				// TODO: don't temporarily leak this goroutine
				// if the program stops itself with the
				// interrupt.
				go func() {
					time.Sleep(mc.KillTimeout)
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
			if status.Signaled() && ctx.Err() != nil {
				return ctx.Err()
			}
			return ExitStatus(status.ExitStatus())
		}
		return ExitStatus(1)
	case *exec.Error:
		// did not start
		fmt.Fprintf(mc.Stderr, "%v\n", err)
		return ExitStatus(127)
	default:
		return err
	}
})

// ModuleOpen is the module responsible for opening a file. It is
// executed for all files that are opened directly by the shell, such as
// in redirects. Files opened by executed programs are not included.
//
// The path parameter is absolute and has been cleaned.
//
// Use a return error of type *os.PathError to have the error printed to
// stderr and the exit status set to 1. If the error is of any other type, the
// interpreter will come to a stop.
//
// TODO: What about stat calls? They are used heavily in the builtin
// test expressions, and also when doing a cd. Should they have a
// separate module?
type ModuleOpen func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error)

func (ModuleOpen) isModule() {}

var DefaultOpen = ModuleOpen(func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	return os.OpenFile(path, flag, perm)
})

func OpenDevImpls(next ModuleOpen) ModuleOpen {
	return func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		mc, _ := FromModuleContext(ctx)
		switch mc.UnixPath(path) {
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
