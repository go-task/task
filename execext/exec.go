package execext

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	"mvdan.cc/sh/interp"
	"mvdan.cc/sh/syntax"
)

var (
	parserPool = sync.Pool{
		New: func() interface{} {
			return syntax.NewParser()
		},
	}

	runnerPool = sync.Pool{
		New: func() interface{} {
			return &interp.Runner{}
		},
	}
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

	parser := parserPool.Get().(*syntax.Parser)
	defer parserPool.Put(parser)

	p, err := parser.Parse(strings.NewReader(opts.Command), "")
	if err != nil {
		return err
	}

	r := runnerPool.Get().(*interp.Runner)
	defer runnerPool.Put(r)

	r.Context = opts.Context
	r.Dir = opts.Dir
	r.Env = opts.Env
	r.Stdin = opts.Stdin
	r.Stdout = opts.Stdout
	r.Stderr = opts.Stderr

	if err = r.Reset(); err != nil {
		return err
	}
	return r.Run(p)
}
