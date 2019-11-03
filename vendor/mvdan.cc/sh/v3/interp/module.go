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

	// KillTimeout is the duration configured by Runner.KillTimeout; refer
	// to its docs for its purpose. It is needed to implement DefaultExec.
	KillTimeout time.Duration
}

// ExecModule is the module responsible for executing a simple command. It is
// executed for all CallExpr nodes where the first argument is neither a
// declared function nor a builtin.
//
// Use a return error of type ExitStatus to set the exit status. A nil error has
// the same effect as ExitStatus(0). If the error is of any other type, the
// interpreter will come to a stop.
type ExecModule func(ctx context.Context, args []string) error

func DefaultExec(ctx context.Context, args []string) error {
	mc, _ := FromModuleContext(ctx)
	path, err := LookPath(mc.Env, args[0])
	if err != nil {
		fmt.Fprintln(mc.Stderr, err)
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

	err = cmd.Start()
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

// OpenModule is the module responsible for opening a file. It is
// executed for all files that are opened directly by the shell, such as
// in redirects. Files opened by executed programs are not included.
//
// The path parameter may be relative to the current directory, which can be
// fetched via FromModuleContext.
//
// Use a return error of type *os.PathError to have the error printed to
// stderr and the exit status set to 1. If the error is of any other type, the
// interpreter will come to a stop.
type OpenModule func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error)

func DefaultOpen(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	mc, _ := FromModuleContext(ctx)
	if !filepath.IsAbs(path) {
		path = filepath.Join(mc.Dir, path)
	}
	return os.OpenFile(path, flag, perm)
}

func OpenDevImpls(next OpenModule) OpenModule {
	return func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
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
