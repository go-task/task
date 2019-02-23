package execext

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/expand"
	"mvdan.cc/sh/interp"
	"mvdan.cc/sh/shell"
	"mvdan.cc/sh/syntax"
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
		interp.Dir(opts.Dir),
		interp.Env(expand.ListEnviron(environ...)),

		interp.Module(interp.DefaultExec),
		interp.Module(interp.OpenDevImpls(interp.DefaultOpen)),

		interp.StdIO(opts.Stdin, opts.Stdout, opts.Stderr),
	)
	if err != nil {
		return err
	}
	return r.Run(ctx, p)
}

// IsExitError returns true the given error is an exis status error
func IsExitError(err error) bool {
	switch err.(type) {
	case interp.ExitStatus, interp.ShellExitStatus:
		return true
	default:
		return false
	}
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
