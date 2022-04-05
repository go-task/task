package execext

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"
)

// RunCommandOptions is the options for the RunCommand func
type RunCommandOptions struct {
	Command string
	Dir     string
	Env     []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

var (
	// ErrNilOptions is returned when a nil options is given
	ErrNilOptions = errors.New("execext: nil options given")
)

// RunCommand runs a shell command
func RunCommand(ctx context.Context, opts *RunCommandOptions) error {
	if opts == nil {
		return ErrNilOptions
	}

	p, err := syntax.NewParser().Parse(strings.NewReader(opts.Command), "")
	if err != nil {
		return err
	}

	environ := opts.Env
	if len(environ) == 0 {
		environ = os.Environ()
	}

	r, err := interp.New(
		interp.Params("-e"),
		interp.Env(expand.ListEnviron(environ...)),
		interp.OpenHandler(openHandler),
		interp.StdIO(opts.Stdin, opts.Stdout, opts.Stderr),
		dirOption(opts.Dir),
	)
	if err != nil {
		return err
	}

	// We used to pass to interp.Runner a context that was cancelled on reception of a
	// OS signal. This caused the Runner to terminate the subprocess abruptly.
	// The correct behavior instead is for us to completely ignore the signal and let
	// the subprocess deal with it. If the subprocess doesn't handle the signal, it will
	// be terminated. If the subprocess does handle the signal, it knows better than us
	// wether it wants to cleanup and terminate or do something different.
	// See https://github.com/go-task/task/issues/458 for details.
	// So now we pass an empty context just to make the API of interp.Runner happy
	return r.Run(context.Background(), p)
}

// IsExitError returns true the given error is an exis status error
func IsExitError(err error) bool {
	if _, ok := interp.IsExitStatus(err); ok {
		return true
	}
	return false
}

// Expand is a helper to mvdan.cc/shell.Fields that returns the first field
// if available.
func Expand(s string) (string, error) {
	s = filepath.ToSlash(s)
	s = strings.Replace(s, " ", `\ `, -1)
	fields, err := shell.Fields(s, nil)
	if err != nil {
		return "", err
	}
	if len(fields) > 0 {
		return fields[0], nil
	}
	return "", nil
}

func openHandler(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	if path == "/dev/null" {
		return devNull{}, nil
	}
	return interp.DefaultOpenHandler()(ctx, path, flag, perm)
}

func dirOption(path string) interp.RunnerOption {
	return func(r *interp.Runner) error {
		err := interp.Dir(path)(r)
		if err == nil {
			return nil
		}

		// If the specified directory doesn't exist, it will be created later.
		// Therefore, even if `interp.Dir` method returns an error, the
		// directory path should be set only when the directory cannot be found.
		if absPath, _ := filepath.Abs(path); absPath != "" {
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				r.Dir = absPath
				return nil
			}
		}

		return err
	}
}
