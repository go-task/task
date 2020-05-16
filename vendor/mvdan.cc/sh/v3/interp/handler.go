// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"mvdan.cc/sh/v3/expand"
)

// HandlerCtx returns HandlerContext value stored in ctx.
// It panics if ctx has no HandlerContext stored.
func HandlerCtx(ctx context.Context) HandlerContext {
	hc, ok := ctx.Value(handlerCtxKey{}).(HandlerContext)
	if !ok {
		panic("interp.HandlerCtx: no HandlerContext in ctx")
	}
	return hc
}

type handlerCtxKey struct{}

// HandlerContext is the data passed to all the handler functions via a context value.
// It contains some of the current state of the Runner.
type HandlerContext struct {
	// Env is a read-only version of the interpreter's environment,
	// including environment variables, global variables, and local function
	// variables.
	Env expand.Environ

	// Dir is the interpreter's current directory.
	Dir string

	// Stdin is the interpreter's current standard input reader.
	Stdin io.Reader
	// Stdout is the interpreter's current standard output writer.
	Stdout io.Writer
	// Stderr is the interpreter's current standard error writer.
	Stderr io.Writer
}

// ExecHandlerFunc is a handler which executes simple command. It is
// called for all CallExpr nodes where the first argument is neither a
// declared function nor a builtin.
//
// Returning nil error sets commands exit status to 0. Other exit statuses
// can be set with NewExitStatus. Any other error will halt an interpreter.
type ExecHandlerFunc func(ctx context.Context, args []string) error

// DefaultExecHandler returns an ExecHandlerFunc used by default.
// It finds binaries in PATH and executes them.
// When context is cancelled, interrupt signal is sent to running processes.
// KillTimeout is a duration to wait before sending kill signal.
// A negative value means that a kill signal will be sent immediately.
// On Windows, the kill signal is always sent immediately,
// because Go doesn't currently support sending Interrupt on Windows.
// Runner.New sets killTimeout to 2 seconds by default.
func DefaultExecHandler(killTimeout time.Duration) ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		hc := HandlerCtx(ctx)
		path, err := LookPath(hc.Env, args[0])
		if err != nil {
			fmt.Fprintln(hc.Stderr, err)
			return NewExitStatus(127)
		}
		cmd := exec.Cmd{
			Path:   path,
			Args:   args,
			Env:    execEnv(hc.Env),
			Dir:    hc.Dir,
			Stdin:  hc.Stdin,
			Stdout: hc.Stdout,
			Stderr: hc.Stderr,
		}

		err = cmd.Start()
		if err == nil {
			if done := ctx.Done(); done != nil {
				go func() {
					<-done

					if killTimeout <= 0 || runtime.GOOS == "windows" {
						_ = cmd.Process.Signal(os.Kill)
						return
					}

					// TODO: don't temporarily leak this goroutine
					// if the program stops itself with the
					// interrupt.
					go func() {
						time.Sleep(killTimeout)
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
				if status.Signaled() {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					return NewExitStatus(uint8(128 + status.Signal()))
				}
				return NewExitStatus(uint8(status.ExitStatus()))
			}
			return NewExitStatus(1)
		case *exec.Error:
			// did not start
			fmt.Fprintf(hc.Stderr, "%v\n", err)
			return NewExitStatus(127)
		default:
			return err
		}
	}
}

func checkStat(dir, file string) (string, error) {
	if !filepath.IsAbs(file) {
		file = filepath.Join(dir, file)
	}
	info, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	m := info.Mode()
	if m.IsDir() {
		return "", fmt.Errorf("is a directory")
	}
	if runtime.GOOS != "windows" && m&0111 == 0 {
		return "", fmt.Errorf("permission denied")
	}
	return file, nil
}

func winHasExt(file string) bool {
	i := strings.LastIndex(file, ".")
	if i < 0 {
		return false
	}
	return strings.LastIndexAny(file, `:\/`) < i
}

func findExecutable(dir, file string, exts []string) (string, error) {
	if len(exts) == 0 {
		// non-windows
		return checkStat(dir, file)
	}
	if winHasExt(file) {
		if file, err := checkStat(dir, file); err == nil {
			return file, nil
		}
	}
	for _, e := range exts {
		f := file + e
		if f, err := checkStat(dir, f); err == nil {
			return f, nil
		}
	}
	return "", fmt.Errorf("not found")
}

func driveLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// splitList is like filepath.SplitList, but always using the unix path
// list separator ':'. On Windows, it also makes sure not to split
// [A-Z]:[/\].
func splitList(path string) []string {
	if path == "" {
		return []string{""}
	}
	list := strings.Split(path, ":")
	if runtime.GOOS != "windows" {
		return list
	}
	// join "C", "/foo" into "C:/foo"
	var fixed []string
	for i := 0; i < len(list); i++ {
		s := list[i]
		switch {
		case len(s) != 1, !driveLetter(s[0]):
		case i+1 >= len(list):
			// last element
		case strings.IndexAny(list[i+1], `/\`) != 0:
			// next element doesn't start with / or \
		default:
			fixed = append(fixed, s+":"+list[i+1])
			i++
			continue
		}
		fixed = append(fixed, s)
	}
	return fixed
}

// LookPath is similar to os/exec.LookPath, with the difference that it uses the
// provided environment. env is used to fetch relevant environment variables
// such as PWD and PATH.
//
// If no error is returned, the returned path must be valid.
func LookPath(env expand.Environ, file string) (string, error) {
	pathList := splitList(env.Get("PATH").String())
	chars := `/`
	if runtime.GOOS == "windows" {
		chars = `:\/`
		// so that "foo" always tries "./foo"
		pathList = append([]string{"."}, pathList...)
	}
	exts := pathExts(env)
	dir := env.Get("PWD").String()
	if strings.ContainsAny(file, chars) {
		return findExecutable(dir, file, exts)
	}
	for _, elem := range pathList {
		var path string
		switch elem {
		case "", ".":
			// otherwise "foo" won't be "./foo"
			path = "." + string(filepath.Separator) + file
		default:
			path = filepath.Join(elem, file)
		}
		if f, err := findExecutable(dir, path, exts); err == nil {
			return f, nil
		}
	}
	return "", fmt.Errorf("%q: executable file not found in $PATH", file)
}

func pathExts(env expand.Environ) []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	pathext := env.Get("PATHEXT").String()
	if pathext == "" {
		return []string{".com", ".exe", ".bat", ".cmd"}
	}
	var exts []string
	for _, e := range strings.Split(strings.ToLower(pathext), `;`) {
		if e == "" {
			continue
		}
		if e[0] != '.' {
			e = "." + e
		}
		exts = append(exts, e)
	}
	return exts
}

// OpenHandlerFunc is a handler which opens files. It is
// called for all files that are opened directly by the shell, such as
// in redirects. Files opened by executed programs are not included.
//
// The path parameter may be relative to the current directory, which can be
// fetched via HandlerCtx.
//
// Use a return error of type *os.PathError to have the error printed to
// stderr and the exit status set to 1. If the error is of any other type, the
// interpreter will come to a stop.
type OpenHandlerFunc func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error)

// DefaultOpenHandler returns an OpenHandlerFunc used by default. It uses os.OpenFile to open files.
func DefaultOpenHandler() OpenHandlerFunc {
	return func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		mc := HandlerCtx(ctx)
		if !filepath.IsAbs(path) {
			path = filepath.Join(mc.Dir, path)
		}
		return os.OpenFile(path, flag, perm)
	}
}
