// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

// Package interp implements an interpreter that executes shell
// programs. It aims to support POSIX, but its support is not complete
// yet. It also supports some Bash features.
package interp

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

// A Runner interprets shell programs. It can be reused, but it is not safe for
// concurrent use. You should typically use New to build a new Runner.
//
// Note that writes to Stdout and Stderr may be concurrent if background
// commands are used. If you plan on using an io.Writer implementation that
// isn't safe for concurrent use, consider a workaround like hiding writes
// behind a mutex.
//
// To create a Runner, use New. Runner's exported fields are meant to be
// configured via runner options; once a Runner has been created, the fields
// should be treated as read-only.
type Runner struct {
	// Env specifies the environment of the interpreter, which must be
	// non-nil.
	Env expand.Environ

	// Dir specifies the working directory of the command, which must be an
	// absolute path.
	Dir string

	// Params are the current shell parameters, e.g. from running a shell
	// file or calling a function. Accessible via the $@/$* family of vars.
	Params []string

	// Separate maps - note that bash allows a name to be both a var and a
	// func simultaneously

	Vars  map[string]expand.Variable
	Funcs map[string]*syntax.Stmt

	alias map[string]alias

	// execHandler is a function responsible for executing programs. It must be non-nil.
	execHandler ExecHandlerFunc

	// openHandler is a function responsible for opening files. It must be non-nil.
	openHandler OpenHandlerFunc

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	ecfg *expand.Config
	ectx context.Context // just so that Runner.Subshell can use it again

	// didReset remembers whether the runner has ever been reset. This is
	// used so that Reset is automatically called when running any program
	// or node for the first time on a Runner.
	didReset bool

	usedNew bool

	// rand is used mainly to generate temporary files.
	rand *rand.Rand

	// wgProcSubsts allows waiting for any process substitution sub-shells
	// to finish running.
	wgProcSubsts sync.WaitGroup

	filename string // only if Node was a File

	// like Vars, but local to a func i.e. "local foo=bar"
	funcVars map[string]expand.Variable

	// like Vars, but local to a cmd i.e. "foo=bar prog args..."
	cmdVars map[string]string

	// >0 to break or continue out of N enclosing loops
	breakEnclosing, contnEnclosing int

	inLoop    bool
	inFunc    bool
	inSource  bool
	noErrExit bool

	// track if a sourced script set positional parameters
	sourceSetParams bool

	err       error // current shell exit code or fatal error
	exitShell bool  // whether the shell needs to exit

	// The current and last exit status code. They can only be different if
	// the interpreter is in the middle of running a statement. In that
	// scenario, 'exit' is the status code for the statement being run, and
	// 'lastExit' corresponds to the previous statement that was run.
	exit     int
	lastExit int

	bgShells errgroup.Group

	opts runnerOpts

	origDir    string
	origParams []string
	origOpts   runnerOpts
	origStdin  io.Reader
	origStdout io.Writer
	origStderr io.Writer

	// Most scripts don't use pushd/popd, so make space for the initial PWD
	// without requiring an extra allocation.
	dirStack     []string
	dirBootstrap [1]string

	optState getopts

	// keepRedirs is used so that "exec" can make any redirections
	// apply to the current shell, and not just the command.
	keepRedirs bool

	// So that we can get io.Copy to reuse the same buffer within a runner.
	// For example, this saves an allocation for every shell pipe, since
	// io.PipeReader does not implement io.WriterTo.
	bufCopier bufCopier
}

type alias struct {
	args  []*syntax.Word
	blank bool
}

type bufCopier struct {
	io.Reader
	buf []byte
}

func (r *bufCopier) WriteTo(w io.Writer) (n int64, err error) {
	if r.buf == nil {
		r.buf = make([]byte, 32*1024)
	}
	return io.CopyBuffer(w, r.Reader, r.buf)
}

func (r *Runner) optByFlag(flag string) *bool {
	for i, opt := range &shellOptsTable {
		if opt.flag == flag {
			return &r.opts[i]
		}
	}
	return nil
}

// New creates a new Runner, applying a number of options. If applying any of
// the options results in an error, it is returned.
//
// Any unset options fall back to their defaults. For example, not supplying the
// environment falls back to the process's environment, and not supplying the
// standard output writer means that the output will be discarded.
func New(opts ...RunnerOption) (*Runner, error) {
	r := &Runner{
		usedNew:     true,
		execHandler: DefaultExecHandler(2 * time.Second),
		openHandler: DefaultOpenHandler(),
	}
	r.dirStack = r.dirBootstrap[:0]
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	// Set the default fallbacks, if necessary.
	if r.Env == nil {
		Env(nil)(r)
	}
	if r.Dir == "" {
		if err := Dir("")(r); err != nil {
			return nil, err
		}
	}
	if r.stdout == nil || r.stderr == nil {
		StdIO(r.stdin, r.stdout, r.stderr)(r)
	}
	return r, nil
}

// RunnerOption is a function which can be passed to New to alter Runner behaviour.
// To apply option to existing Runner call it directly,
// for example interp.Params("-e")(runner).
type RunnerOption func(*Runner) error

// Env sets the interpreter's environment. If nil, a copy of the current
// process's environment is used.
func Env(env expand.Environ) RunnerOption {
	return func(r *Runner) error {
		if env == nil {
			env = expand.ListEnviron(os.Environ()...)
		}
		r.Env = env
		return nil
	}
}

// Dir sets the interpreter's working directory. If empty, the process's current
// directory is used.
func Dir(path string) RunnerOption {
	return func(r *Runner) error {
		if path == "" {
			path, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("could not get current dir: %v", err)
			}
			r.Dir = path
			return nil
		}
		path, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("could not get absolute dir: %v", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("could not stat: %v", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
		r.Dir = path
		return nil
	}
}

// Params populates the shell options and parameters. For example, Params("-e",
// "--", "foo") will set the "-e" option and the parameters ["foo"], and
// Params("+e") will unset the "-e" option and leave the parameters untouched.
//
// This is similar to what the interpreter's "set" builtin does.
func Params(args ...string) RunnerOption {
	return func(r *Runner) error {
		onlyFlags := true
		for len(args) > 0 {
			arg := args[0]
			if arg == "" || (arg[0] != '-' && arg[0] != '+') {
				onlyFlags = false
				break
			}
			if arg == "--" {
				onlyFlags = false
				args = args[1:]
				break
			}
			enable := arg[0] == '-'
			var opt *bool
			if flag := arg[1:]; flag == "o" {
				args = args[1:]
				if len(args) == 0 && enable {
					for i, opt := range &shellOptsTable {
						r.printOptLine(opt.name, r.opts[i])
					}
					break
				}
				if len(args) == 0 && !enable {
					for i, opt := range &shellOptsTable {
						setFlag := "+o"
						if r.opts[i] {
							setFlag = "-o"
						}
						r.outf("set %s %s\n", setFlag, opt.name)
					}
					break
				}
				opt = r.optByName(args[0], false)
			} else {
				opt = r.optByFlag(flag)
			}
			if opt == nil {
				return fmt.Errorf("invalid option: %q", arg)
			}
			*opt = enable
			args = args[1:]
		}
		if !onlyFlags {
			// If "--" wasn't given and there were zero arguments,
			// we don't want to override the current parameters.
			r.Params = args

			// Record whether a sourced script sets the parameters.
			if r.inSource {
				r.sourceSetParams = true
			}
		}
		return nil
	}
}

// ExecHandler sets command execution handler. See ExecHandlerFunc for more info.
func ExecHandler(f ExecHandlerFunc) RunnerOption {
	return func(r *Runner) error {
		r.execHandler = f
		return nil
	}
}

// OpenHandler sets file open handler. See OpenHandlerFunc for more info.
func OpenHandler(f OpenHandlerFunc) RunnerOption {
	return func(r *Runner) error {
		r.openHandler = f
		return nil
	}
}

// StdIO configures an interpreter's standard input, standard output, and
// standard error. If out or err are nil, they default to a writer that discards
// the output.
func StdIO(in io.Reader, out, err io.Writer) RunnerOption {
	return func(r *Runner) error {
		r.stdin = in
		if out == nil {
			out = ioutil.Discard
		}
		r.stdout = out
		if err == nil {
			err = ioutil.Discard
		}
		r.stderr = err
		return nil
	}
}

func (r *Runner) optByName(name string, bash bool) *bool {
	if bash {
		for i, optName := range bashOptsTable {
			if optName == name {
				return &r.opts[len(shellOptsTable)+i]
			}
		}
	}
	for i, opt := range &shellOptsTable {
		if opt.name == name {
			return &r.opts[i]
		}
	}
	return nil
}

type runnerOpts [len(shellOptsTable) + len(bashOptsTable)]bool

var shellOptsTable = [...]struct {
	flag, name string
}{
	// sorted alphabetically by name; use a space for the options
	// that have no flag form
	{"a", "allexport"},
	{"e", "errexit"},
	{"n", "noexec"},
	{"f", "noglob"},
	{"u", "nounset"},
	{" ", "pipefail"},
}

var bashOptsTable = [...]string{
	// sorted alphabetically by name
	"expand_aliases",
	"globstar",
}

// To access the shell options arrays without a linear search when we
// know which option we're after at compile time. First come the shell options,
// then the bash options.
const (
	optAllExport = iota
	optErrExit
	optNoExec
	optNoGlob
	optNoUnset
	optPipeFail

	optExpandAliases
	optGlobStar
)

// Reset returns a runner to its initial state, right before the first call to
// Run or Reset.
//
// Typically, this function only needs to be called if a runner is reused to run
// multiple programs non-incrementally. Not calling Reset between each run will
// mean that the shell state will be kept, including variables, options, and the
// current directory.
func (r *Runner) Reset() {
	if !r.usedNew {
		panic("use interp.New to construct a Runner")
	}
	if !r.didReset {
		r.origDir = r.Dir
		r.origParams = r.Params
		r.origOpts = r.opts
		r.origStdin = r.stdin
		r.origStdout = r.stdout
		r.origStderr = r.stderr
	}
	// reset the internal state
	*r = Runner{
		Env:         r.Env,
		execHandler: r.execHandler,
		openHandler: r.openHandler,

		// These can be set by functions like Dir or Params, but
		// builtins can overwrite them; reset the fields to whatever the
		// constructor set up.
		Dir:    r.origDir,
		Params: r.origParams,
		opts:   r.origOpts,
		stdin:  r.origStdin,
		stdout: r.origStdout,
		stderr: r.origStderr,

		origDir:    r.origDir,
		origParams: r.origParams,
		origOpts:   r.origOpts,
		origStdin:  r.origStdin,
		origStdout: r.origStdout,
		origStderr: r.origStderr,

		// emptied below, to reuse the space
		Vars:      r.Vars,
		cmdVars:   r.cmdVars,
		dirStack:  r.dirStack[:0],
		usedNew:   r.usedNew,
		bufCopier: r.bufCopier,
	}
	if r.Vars == nil {
		r.Vars = make(map[string]expand.Variable)
	} else {
		for k := range r.Vars {
			delete(r.Vars, k)
		}
	}
	if r.cmdVars == nil {
		r.cmdVars = make(map[string]string)
	} else {
		for k := range r.cmdVars {
			delete(r.cmdVars, k)
		}
	}
	if vr := r.Env.Get("HOME"); !vr.IsSet() {
		home, _ := os.UserHomeDir()
		r.Vars["HOME"] = expand.Variable{Kind: expand.String, Str: home}
	}
	r.Vars["UID"] = expand.Variable{
		Kind:     expand.String,
		ReadOnly: true,
		Str:      strconv.Itoa(os.Getuid()),
	}
	r.Vars["PWD"] = expand.Variable{Kind: expand.String, Str: r.Dir}
	r.Vars["IFS"] = expand.Variable{Kind: expand.String, Str: " \t\n"}
	r.Vars["OPTIND"] = expand.Variable{Kind: expand.String, Str: "1"}

	if runtime.GOOS == "windows" {
		// convert $PATH to a unix path list
		path := r.Env.Get("PATH").String()
		path = strings.Join(filepath.SplitList(path), ":")
		r.Vars["PATH"] = expand.Variable{Kind: expand.String, Str: path}
	}

	r.dirStack = append(r.dirStack, r.Dir)
	r.didReset = true
	r.bufCopier.Reader = nil
}

// exitStatus is a non-zero status code resulting from running a shell node.
type exitStatus uint8

func (s exitStatus) Error() string { return fmt.Sprintf("exit status %d", s) }

// NewExitStatus creates an error which contains the specified exit status code.
func NewExitStatus(status uint8) error {
	return exitStatus(status)
}

// IsExitStatus checks whether error contains an exit status and returns it.
func IsExitStatus(err error) (status uint8, ok bool) {
	var s exitStatus
	if xerrors.As(err, &s) {
		return uint8(s), true
	}
	return 0, false
}

// Run interprets a node, which can be a *File, *Stmt, or Command. If a non-nil
// error is returned, it will typically contain commands exit status,
// which can be retrieved with IsExitStatus.
//
// Run can be called multiple times synchronously to interpret programs
// incrementally. To reuse a Runner without keeping the internal shell state,
// call Reset.
func (r *Runner) Run(ctx context.Context, node syntax.Node) error {
	if !r.didReset {
		r.Reset()
	}
	r.fillExpandConfig(ctx)
	r.err = nil
	r.exitShell = false
	r.filename = ""
	switch x := node.(type) {
	case *syntax.File:
		r.filename = x.Name
		r.stmts(ctx, x.Stmts)
	case *syntax.Stmt:
		r.stmt(ctx, x)
	case syntax.Command:
		r.cmd(ctx, x)
	default:
		return fmt.Errorf("node can only be File, Stmt, or Command: %T", x)
	}
	if r.exit != 0 {
		r.setErr(NewExitStatus(uint8(r.exit)))
	}
	return r.err
}

// Exited reports whether the last Run call should exit an entire shell. This
// can be triggered by the "exit" built-in command, for example.
//
// Note that this state is overwritten at every Run call, so it should be
// checked immediately after each Run call.
func (r *Runner) Exited() bool {
	return r.exitShell
}

// Subshell makes a copy of the given Runner, suitable for use concurrently
// with the original.  The copy will have the same environment, including
// variables and functions, but they can all be modified without affecting the
// original.
//
// Subshell is not safe to use concurrently with Run.  Orchestrating this is
// left up to the caller; no locking is performed.
//
// To replace e.g. stdin/out/err, do StdIO(r.stdin, r.stdout, r.stderr)(r) on
// the copy.
func (r *Runner) Subshell() *Runner {
	// Keep in sync with the Runner type. Manually copy fields, to not copy
	// sensitive ones like errgroup.Group, and to do deep copies of slices.
	r2 := &Runner{
		Env:         r.Env,
		Dir:         r.Dir,
		Params:      r.Params,
		execHandler: r.execHandler,
		openHandler: r.openHandler,
		stdin:       r.stdin,
		stdout:      r.stdout,
		stderr:      r.stderr,
		filename:    r.filename,
		opts:        r.opts,
		usedNew:     r.usedNew,
		exit:        r.exit,
		lastExit:    r.lastExit,

		origStdout: r.origStdout, // used for process substitutions
	}
	r2.Vars = make(map[string]expand.Variable, len(r.Vars))
	for k, v := range r.Vars {
		v2 := v
		// Make deeper copies of List and Map, but ensure that they remain nil
		// if they are nil in v.
		v2.List = append([]string(nil), v.List...)
		if v.Map != nil {
			v2.Map = make(map[string]string, len(v.Map))
			for k, v := range v.Map {
				v2.Map[k] = v
			}
		}
		r2.Vars[k] = v2
	}
	r2.funcVars = make(map[string]expand.Variable, len(r.funcVars))
	for k, v := range r.funcVars {
		r2.funcVars[k] = v
	}
	r2.cmdVars = make(map[string]string, len(r.cmdVars))
	for k, v := range r.cmdVars {
		r2.cmdVars[k] = v
	}
	r2.Funcs = make(map[string]*syntax.Stmt, len(r.Funcs))
	for k, v := range r.Funcs {
		r2.Funcs[k] = v
	}
	if l := len(r.alias); l > 0 {
		r2.alias = make(map[string]alias, l)
		for k, v := range r.alias {
			r2.alias[k] = v
		}
	}

	r2.dirStack = append(r2.dirBootstrap[:0], r.dirStack...)
	r2.fillExpandConfig(r.ectx)
	r2.didReset = true
	return r2
}
