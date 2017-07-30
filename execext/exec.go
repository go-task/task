package execext

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/mvdan/sh/interp"
	"github.com/mvdan/sh/syntax"
)

// RunCommandOptions is the options for the RunCommand func
type RunCommandOptions struct {
	Context context.Context
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
func RunCommand(opts *RunCommandOptions) error {
	if opts == nil {
		return ErrNilOptions
	}

	p, err := syntax.NewParser().Parse(strings.NewReader(opts.Command), "")
	if err != nil {
		return err
	}

	r := interp.Runner{
		Context: opts.Context,
		Node:    p,
		Dir:     opts.Dir,
		Env:     opts.Env,
		Stdin:   opts.Stdin,
		Stdout:  opts.Stdout,
		Stderr:  opts.Stderr,
	}
	return r.Run()
}
