package execext

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/moreinterp/coreutils"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"github.com/go-task/task/v3/errors"
)

// ErrNilOptions is returned when a nil options is given
var ErrNilOptions = errors.New("execext: nil options given")

// RunCommandOptions is the options for the [RunCommand] func.
type RunCommandOptions struct {
	Command   string
	Dir       string
	Env       []string
	PosixOpts []string
	BashOpts  []string
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
}

// RunCommand runs a shell command
func RunCommand(ctx context.Context, opts *RunCommandOptions) error {
	if opts == nil {
		return ErrNilOptions
	}

	// Set "-e" or "errexit" by default
	opts.PosixOpts = append(opts.PosixOpts, "e")

	// Format POSIX options into a slice that mvdan/sh understands
	var params []string
	for _, opt := range opts.PosixOpts {
		if len(opt) == 1 {
			params = append(params, fmt.Sprintf("-%s", opt))
		} else {
			params = append(params, "-o")
			params = append(params, opt)
		}
	}

	environ := opts.Env
	if len(environ) == 0 {
		environ = os.Environ()
	}

	r, err := interp.New(
		interp.Params(params...),
		interp.Env(expand.ListEnviron(environ...)),
		interp.ExecHandlers(execHandlers()...),
		interp.OpenHandler(openHandler),
		interp.StdIO(opts.Stdin, opts.Stdout, opts.Stderr),
		dirOption(opts.Dir),
	)
	if err != nil {
		return err
	}

	parser := syntax.NewParser()

	// Run any shopt commands
	if len(opts.BashOpts) > 0 {
		shoptCmdStr := fmt.Sprintf("shopt -s %s", strings.Join(opts.BashOpts, " "))
		shoptCmd, err := parser.Parse(strings.NewReader(shoptCmdStr), "")
		if err != nil {
			return err
		}
		if err := r.Run(ctx, shoptCmd); err != nil {
			return err
		}
	}

	// Run the user-defined command
	p, err := parser.Parse(strings.NewReader(opts.Command), "")
	if err != nil {
		return err
	}
	return r.Run(ctx, p)
}

func escape(s string) string {
	s = filepath.ToSlash(s)
	s = strings.ReplaceAll(s, " ", `\ `)
	s = strings.ReplaceAll(s, "&", `\&`)
	s = strings.ReplaceAll(s, "(", `\(`)
	s = strings.ReplaceAll(s, ")", `\)`)
	return s
}

// ExpandLiteral is a wrapper around [expand.Literal]. It will escape the input
// string, expand any shell symbols (such as '~') and resolve any environment
// variables.
func ExpandLiteral(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	p := syntax.NewParser()
	word, err := p.Document(strings.NewReader(s))
	if err != nil {
		return "", err
	}
	cfg := &expand.Config{
		Env:      expand.FuncEnviron(os.Getenv),
		ReadDir2: os.ReadDir,
		GlobStar: true,
	}
	return expand.Literal(cfg, word)
}

// ExpandFields is a wrapper around [expand.Fields]. It will escape the input
// string, expand any shell symbols (such as '~') and resolve any environment
// variables. It also expands brace expressions ({a.b}) and globs (*/**) and
// returns the results as a list of strings.
func ExpandFields(s string) ([]string, error) {
	s = escape(s)
	p := syntax.NewParser()
	var words []*syntax.Word
	err := p.Words(strings.NewReader(s), func(w *syntax.Word) bool {
		words = append(words, w)
		return true
	})
	if err != nil {
		return nil, err
	}
	cfg := &expand.Config{
		Env:      expand.FuncEnviron(os.Getenv),
		ReadDir2: os.ReadDir,
		GlobStar: true,
		NullGlob: true,
	}
	return expand.Fields(cfg, words...)
}

func execHandlers() (handlers []func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc) {
	if useGoCoreUtils {
		handlers = append(handlers, coreutils.ExecHandler)
	}
	return handlers
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
