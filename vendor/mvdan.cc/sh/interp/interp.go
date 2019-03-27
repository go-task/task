// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"mvdan.cc/sh/expand"
	"mvdan.cc/sh/syntax"
)

// New creates a new Runner, applying a number of options. If applying any of
// the options results in an error, it is returned.
//
// Any unset options fall back to their defaults. For example, not supplying the
// environment falls back to the process's environment, and not supplying the
// standard output writer means that the output will be discarded.
func New(opts ...func(*Runner) error) (*Runner, error) {
	r := &Runner{usedNew: true}
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
	if r.Exec == nil {
		Module(ModuleExec(nil))(r)
	}
	if r.Open == nil {
		Module(ModuleOpen(nil))(r)
	}
	if r.Stdout == nil || r.Stderr == nil {
		StdIO(r.Stdin, r.Stdout, r.Stderr)(r)
	}
	return r, nil
}

func (r *Runner) fillExpandConfig(ctx context.Context) {
	r.ectx = ctx
	r.ecfg = &expand.Config{
		Env: expandEnv{r},
		CmdSubst: func(w io.Writer, cs *syntax.CmdSubst) error {
			switch len(cs.Stmts) {
			case 0: // nothing to do
				return nil
			case 1: // $(<file)
				word := catShortcutArg(cs.Stmts[0])
				if word == nil {
					break
				}
				path := r.literal(word)
				f, err := r.open(ctx, r.relPath(path), os.O_RDONLY, 0, true)
				if err != nil {
					return err
				}
				_, err = io.Copy(w, f)
				return err
			}
			r2 := r.sub()
			r2.Stdout = w
			r2.stmts(ctx, cs.StmtList)
			return r2.err
		},
		ReadDir: ioutil.ReadDir,
	}
	r.updateExpandOpts()
}

// catShortcutArg checks if a statement is of the form "$(<file)". The redirect
// word is returned if there's a match, and nil otherwise.
func catShortcutArg(stmt *syntax.Stmt) *syntax.Word {
	if stmt.Cmd != nil || stmt.Negated || stmt.Background || stmt.Coprocess {
		return nil
	}
	if len(stmt.Redirs) != 1 {
		return nil
	}
	redir := stmt.Redirs[0]
	if redir.Op != syntax.RdrIn {
		return nil
	}
	return redir.Word
}

func (r *Runner) updateExpandOpts() {
	r.ecfg.NoGlob = r.opts[optNoGlob]
	r.ecfg.GlobStar = r.opts[optGlobStar]
}

func (r *Runner) expandErr(err error) {
	switch err := err.(type) {
	case nil:
	case expand.UnsetParameterError:
		r.errf("%s\n", err.Message)
		r.exit = 1
		r.setErr(ShellExitStatus(r.exit))
	default:
		r.setErr(err)
		r.exit = 1
	}
}

func (r *Runner) arithm(expr syntax.ArithmExpr) int {
	n, err := expand.Arithm(r.ecfg, expr)
	r.expandErr(err)
	return n
}

func (r *Runner) fields(words ...*syntax.Word) []string {
	strs, err := expand.Fields(r.ecfg, words...)
	r.expandErr(err)
	return strs
}

func (r *Runner) literal(word *syntax.Word) string {
	str, err := expand.Literal(r.ecfg, word)
	r.expandErr(err)
	return str
}

func (r *Runner) document(word *syntax.Word) string {
	str, err := expand.Document(r.ecfg, word)
	r.expandErr(err)
	return str
}

func (r *Runner) pattern(word *syntax.Word) string {
	str, err := expand.Pattern(r.ecfg, word)
	r.expandErr(err)
	return str
}

// expandEnv exposes Runner's variables to the expand package.
type expandEnv struct {
	r *Runner
}

func (e expandEnv) Get(name string) expand.Variable {
	return e.r.lookupVar(name)
}
func (e expandEnv) Set(name string, vr expand.Variable) {
	e.r.setVarInternal(name, vr)
}
func (e expandEnv) Each(fn func(name string, vr expand.Variable) bool) {
	e.r.Env.Each(fn)
	for name, vr := range e.r.Vars {
		if !fn(name, vr) {
			return
		}
	}
}

// Env sets the interpreter's environment. If nil, a copy of the current
// process's environment is used.
func Env(env expand.Environ) func(*Runner) error {
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
func Dir(path string) func(*Runner) error {
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
// "--", "foo") will set the "-e" option and the parameters ["foo"].
//
// This is similar to what the interpreter's "set" builtin does.
func Params(args ...string) func(*Runner) error {
	return func(r *Runner) error {
		for len(args) > 0 {
			arg := args[0]
			if arg == "" || (arg[0] != '-' && arg[0] != '+') {
				break
			}
			if arg == "--" {
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
		r.Params = args
		return nil
	}
}

type ModuleFunc interface {
	isModule()
}

// Module sets an interpreter module, which can be ModuleExec or ModuleOpen. If
// the value is nil, the default module implementation is used.
func Module(mod ModuleFunc) func(*Runner) error {
	return func(r *Runner) error {
		switch mod := mod.(type) {
		case ModuleExec:
			if mod == nil {
				mod = DefaultExec
			}
			r.Exec = mod
		case ModuleOpen:
			if mod == nil {
				mod = DefaultOpen
			}
			r.Open = mod
		default:
			return fmt.Errorf("unknown module type: %T", mod)
		}
		return nil
	}
}

// StdIO configures an interpreter's standard input, standard output, and
// standard error. If out or err are nil, they default to a writer that discards
// the output.
func StdIO(in io.Reader, out, err io.Writer) func(*Runner) error {
	return func(r *Runner) error {
		r.Stdin = in
		if out == nil {
			out = ioutil.Discard
		}
		r.Stdout = out
		if err == nil {
			err = ioutil.Discard
		}
		r.Stderr = err
		return nil
	}
}

// A Runner interprets shell programs. It can be reused, but it is not safe for
// concurrent use. You should typically use New to build a new Runner.
//
// Note that writes to Stdout and Stderr may be concurrent if background
// commands are used. If you plan on using an io.Writer implementation that
// isn't safe for concurrent use, consider a workaround like hiding writes
// behind a mutex.
//
// To create a Runner, use New.
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

	// Exec is the module responsible for executing programs. It must be
	// non-nil.
	Exec ModuleExec
	// Open is the module responsible for opening files. It must be non-nil.
	Open ModuleOpen

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// Separate maps - note that bash allows a name to be both a var and a
	// func simultaneously

	Vars  map[string]expand.Variable
	Funcs map[string]*syntax.Stmt

	ecfg *expand.Config
	ectx context.Context // just so that Runner.Sub can use it again

	// didReset remembers whether the runner has ever been reset. This is
	// used so that Reset is automatically called when running any program
	// or node for the first time on a Runner.
	didReset bool

	usedNew bool

	filename string // only if Node was a File

	// like Vars, but local to a func i.e. "local foo=bar"
	funcVars map[string]expand.Variable

	// like Vars, but local to a cmd i.e. "foo=bar prog args..."
	cmdVars map[string]string

	// >0 to break or continue out of N enclosing loops
	breakEnclosing, contnEnclosing int

	inLoop   bool
	inFunc   bool
	inSource bool

	err  error // current shell exit code or fatal error
	exit int   // current (last) exit status code

	bgShells errgroup.Group

	opts [len(shellOptsTable) + len(bashOptsTable)]bool

	dirStack []string

	optState getopts

	// keepRedirs is used so that "exec" can make any redirections
	// apply to the current shell, and not just the command.
	keepRedirs bool

	// KillTimeout holds how much time the interpreter will wait for a
	// program to stop after being sent an interrupt signal, after
	// which a kill signal will be sent. This process will happen when the
	// interpreter's context is cancelled.
	//
	// The zero value will default to 2 seconds.
	//
	// A negative value means that a kill signal will be sent immediately.
	//
	// On Windows, the kill signal is always sent immediately,
	// because Go doesn't currently support sending Interrupt on Windows.
	KillTimeout time.Duration
}

func (r *Runner) optByFlag(flag string) *bool {
	for i, opt := range &shellOptsTable {
		if opt.flag == flag {
			return &r.opts[i]
		}
	}
	return nil
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

	optGlobStar
)

// Reset empties the runner state and sets any exported fields with zero values
// to their default values.
//
// Typically, this function only needs to be called if a runner is reused to run
// multiple programs non-incrementally. Not calling Reset between each run will
// mean that the shell state will be kept, including variables and options.
func (r *Runner) Reset() {
	if !r.usedNew {
		panic("use interp.New to construct a Runner")
	}
	// reset the internal state
	*r = Runner{
		Env:         r.Env,
		Dir:         r.Dir,
		Params:      r.Params,
		Stdin:       r.Stdin,
		Stdout:      r.Stdout,
		Stderr:      r.Stderr,
		Exec:        r.Exec,
		Open:        r.Open,
		KillTimeout: r.KillTimeout,
		opts:        r.opts,

		// emptied below, to reuse the space
		Vars:     r.Vars,
		cmdVars:  r.cmdVars,
		dirStack: r.dirStack[:0],
		usedNew:  r.usedNew,
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
		u, _ := user.Current()
		r.Vars["HOME"] = expand.Variable{Value: u.HomeDir}
	}
	r.Vars["PWD"] = expand.Variable{Value: r.Dir}
	r.Vars["IFS"] = expand.Variable{Value: " \t\n"}
	r.Vars["OPTIND"] = expand.Variable{Value: "1"}

	if runtime.GOOS == "windows" {
		// convert $PATH to a unix path list
		path := r.Env.Get("PATH").String()
		path = strings.Join(filepath.SplitList(path), ":")
		r.Vars["PATH"] = expand.Variable{Value: path}
	}

	r.dirStack = append(r.dirStack, r.Dir)
	if r.KillTimeout == 0 {
		r.KillTimeout = 2 * time.Second
	}
	r.didReset = true
}

func (r *Runner) modCtx(ctx context.Context) context.Context {
	mc := ModuleCtx{
		Dir:         r.Dir,
		Stdin:       r.Stdin,
		Stdout:      r.Stdout,
		Stderr:      r.Stderr,
		KillTimeout: r.KillTimeout,
	}
	oenv := overlayEnviron{
		parent: r.Env,
		values: make(map[string]expand.Variable),
	}
	for name, vr := range r.Vars {
		oenv.Set(name, vr)
	}
	for name, vr := range r.funcVars {
		oenv.Set(name, vr)
	}
	for name, value := range r.cmdVars {
		oenv.Set(name, expand.Variable{Exported: true, Value: value})
	}
	mc.Env = oenv
	return context.WithValue(ctx, moduleCtxKey{}, mc)
}

// ShellExitStatus exits the shell with a status code.
type ShellExitStatus uint8

func (s ShellExitStatus) Error() string { return fmt.Sprintf("exit status %d", s) }

// ExitStatus is a non-zero status code resulting from running a shell node.
type ExitStatus uint8

func (s ExitStatus) Error() string { return fmt.Sprintf("exit status %d", s) }

func (r *Runner) setErr(err error) {
	if r.err == nil {
		r.err = err
	}
}

// Run interprets a node, which can be a *File, *Stmt, or Command. If a non-nil
// error is returned, it will typically be of type ExitStatus or
// ShellExitStatus.
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
	r.filename = ""
	switch x := node.(type) {
	case *syntax.File:
		r.filename = x.Name
		r.stmts(ctx, x.StmtList)
	case *syntax.Stmt:
		r.stmt(ctx, x)
	case syntax.Command:
		r.cmd(ctx, x)
	default:
		return fmt.Errorf("node can only be File, Stmt, or Command: %T", x)
	}
	if r.exit > 0 {
		r.setErr(ExitStatus(r.exit))
	}
	return r.err
}

func (r *Runner) out(s string) {
	io.WriteString(r.Stdout, s)
}

func (r *Runner) outf(format string, a ...interface{}) {
	fmt.Fprintf(r.Stdout, format, a...)
}

func (r *Runner) errf(format string, a ...interface{}) {
	fmt.Fprintf(r.Stderr, format, a...)
}

func (r *Runner) stop(ctx context.Context) bool {
	if r.err != nil {
		return true
	}
	if err := ctx.Err(); err != nil {
		r.err = err
		return true
	}
	if r.opts[optNoExec] {
		return true
	}
	return false
}

func (r *Runner) stmt(ctx context.Context, st *syntax.Stmt) {
	if r.stop(ctx) {
		return
	}
	if st.Background {
		r2 := r.sub()
		st2 := *st
		st2.Background = false
		r.bgShells.Go(func() error {
			return r2.Run(ctx, &st2)
		})
	} else {
		r.stmtSync(ctx, st)
	}
}

func (r *Runner) stmtSync(ctx context.Context, st *syntax.Stmt) {
	oldIn, oldOut, oldErr := r.Stdin, r.Stdout, r.Stderr
	for _, rd := range st.Redirs {
		cls, err := r.redir(ctx, rd)
		if err != nil {
			r.exit = 1
			return
		}
		if cls != nil {
			defer cls.Close()
		}
	}
	if st.Cmd == nil {
		r.exit = 0
	} else {
		r.cmd(ctx, st.Cmd)
	}
	if st.Negated {
		r.exit = oneIf(r.exit == 0)
	}
	if r.exit != 0 && r.opts[optErrExit] {
		r.setErr(ShellExitStatus(r.exit))
	}
	if !r.keepRedirs {
		r.Stdin, r.Stdout, r.Stderr = oldIn, oldOut, oldErr
	}
}

func (r *Runner) sub() *Runner {
	// Keep in sync with the Runner type. Manually copy fields, to not copy
	// sensitive ones like errgroup.Group, and to do deep copies of slices.
	r2 := &Runner{
		Env:         r.Env,
		Dir:         r.Dir,
		Params:      r.Params,
		Exec:        r.Exec,
		Open:        r.Open,
		Stdin:       r.Stdin,
		Stdout:      r.Stdout,
		Stderr:      r.Stderr,
		Funcs:       r.Funcs,
		KillTimeout: r.KillTimeout,
		filename:    r.filename,
		opts:        r.opts,
	}
	r2.Vars = make(map[string]expand.Variable, len(r.Vars))
	for k, v := range r.Vars {
		r2.Vars[k] = v
	}
	r2.funcVars = make(map[string]expand.Variable, len(r.funcVars))
	for k, v := range r.funcVars {
		r2.funcVars[k] = v
	}
	r2.cmdVars = make(map[string]string, len(r.cmdVars))
	for k, v := range r.cmdVars {
		r2.cmdVars[k] = v
	}
	r2.dirStack = append([]string(nil), r.dirStack...)
	r2.fillExpandConfig(r.ectx)
	r2.didReset = true
	return r2
}

func (r *Runner) cmd(ctx context.Context, cm syntax.Command) {
	if r.stop(ctx) {
		return
	}
	switch x := cm.(type) {
	case *syntax.Block:
		r.stmts(ctx, x.StmtList)
	case *syntax.Subshell:
		r2 := r.sub()
		r2.stmts(ctx, x.StmtList)
		r.exit = r2.exit
		r.setErr(r2.err)
	case *syntax.CallExpr:
		fields := r.fields(x.Args...)
		if len(fields) == 0 {
			for _, as := range x.Assigns {
				vr := r.lookupVar(as.Name.Value)
				vr.Value = r.assignVal(as, "")
				r.setVar(as.Name.Value, as.Index, vr)
			}
			break
		}
		for _, as := range x.Assigns {
			val := r.assignVal(as, "")
			// we know that inline vars must be strings
			r.cmdVars[as.Name.Value] = val.(string)
		}
		r.call(ctx, x.Args[0].Pos(), fields)
		// cmdVars can be nuked here, as they are never useful
		// again once we nest into further levels of inline
		// vars.
		for k := range r.cmdVars {
			delete(r.cmdVars, k)
		}
	case *syntax.BinaryCmd:
		switch x.Op {
		case syntax.AndStmt:
			r.stmt(ctx, x.X)
			if r.exit == 0 {
				r.stmt(ctx, x.Y)
			}
		case syntax.OrStmt:
			r.stmt(ctx, x.X)
			if r.exit != 0 {
				r.stmt(ctx, x.Y)
			}
		case syntax.Pipe, syntax.PipeAll:
			pr, pw := io.Pipe()
			r2 := r.sub()
			r2.Stdout = pw
			if x.Op == syntax.PipeAll {
				r2.Stderr = pw
			} else {
				r2.Stderr = r.Stderr
			}
			r.Stdin = pr
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				r2.stmt(ctx, x.X)
				pw.Close()
				wg.Done()
			}()
			r.stmt(ctx, x.Y)
			pr.Close()
			wg.Wait()
			if r.opts[optPipeFail] && r2.exit > 0 && r.exit == 0 {
				r.exit = r2.exit
			}
			r.setErr(r2.err)
		}
	case *syntax.IfClause:
		r.stmts(ctx, x.Cond)
		if r.exit == 0 {
			r.stmts(ctx, x.Then)
			break
		}
		r.exit = 0
		r.stmts(ctx, x.Else)
	case *syntax.WhileClause:
		for !r.stop(ctx) {
			r.stmts(ctx, x.Cond)
			stop := (r.exit == 0) == x.Until
			r.exit = 0
			if stop || r.loopStmtsBroken(ctx, x.Do) {
				break
			}
		}
	case *syntax.ForClause:
		switch y := x.Loop.(type) {
		case *syntax.WordIter:
			name := y.Name.Value
			items := r.Params // for i; do ...
			if y.InPos.IsValid() {
				items = r.fields(y.Items...) // for i in ...; do ...
			}
			for _, field := range items {
				r.setVarString(name, field)
				if r.loopStmtsBroken(ctx, x.Do) {
					break
				}
			}
		case *syntax.CStyleLoop:
			r.arithm(y.Init)
			for r.arithm(y.Cond) != 0 {
				if r.loopStmtsBroken(ctx, x.Do) {
					break
				}
				r.arithm(y.Post)
			}
		}
	case *syntax.FuncDecl:
		r.setFunc(x.Name.Value, x.Body)
	case *syntax.ArithmCmd:
		r.exit = oneIf(r.arithm(x.X) == 0)
	case *syntax.LetClause:
		var val int
		for _, expr := range x.Exprs {
			val = r.arithm(expr)
		}
		r.exit = oneIf(val == 0)
	case *syntax.CaseClause:
		str := r.literal(x.Word)
		for _, ci := range x.Items {
			for _, word := range ci.Patterns {
				pattern := r.pattern(word)
				if match(pattern, str) {
					r.stmts(ctx, ci.StmtList)
					return
				}
			}
		}
	case *syntax.TestClause:
		r.exit = 0
		if r.bashTest(ctx, x.X, false) == "" && r.exit == 0 {
			// to preserve exit status code 2 for regex errors, etc
			r.exit = 1
		}
	case *syntax.DeclClause:
		local, global := false, false
		var modes []string
		valType := ""
		switch x.Variant.Value {
		case "declare":
			// When used in a function, "declare" acts as "local"
			// unless the "-g" option is used.
			local = r.inFunc
		case "local":
			if !r.inFunc {
				r.errf("local: can only be used in a function\n")
				r.exit = 1
				return
			}
			local = true
		case "export":
			modes = append(modes, "-x")
		case "readonly":
			modes = append(modes, "-r")
		case "nameref":
			modes = append(modes, "-n")
		}
		for _, opt := range x.Opts {
			switch s := r.literal(opt); s {
			case "-x", "-r", "-n":
				modes = append(modes, s)
			case "-a", "-A":
				valType = s
			case "-g":
				global = true
			default:
				r.errf("declare: invalid option %q\n", s)
				r.exit = 2
				return
			}
		}
		for _, as := range x.Assigns {
			for _, as := range r.flattenAssign(as) {
				name := as.Name.Value
				if !syntax.ValidName(name) {
					r.errf("declare: invalid name %q\n", name)
					r.exit = 1
					return
				}
				vr := r.lookupVar(as.Name.Value)
				vr.Value = r.assignVal(as, valType)
				if global {
					vr.Local = false
				} else if local {
					vr.Local = true
				}
				for _, mode := range modes {
					switch mode {
					case "-x":
						vr.Exported = true
					case "-r":
						vr.ReadOnly = true
					case "-n":
						vr.NameRef = true
					}
				}
				r.setVar(name, as.Index, vr)
			}
		}
	case *syntax.TimeClause:
		start := time.Now()
		if x.Stmt != nil {
			r.stmt(ctx, x.Stmt)
		}
		format := "%s\t%s\n"
		if x.PosixFormat {
			format = "%s %s\n"
		} else {
			r.outf("\n")
		}
		real := time.Since(start)
		r.outf(format, "real", elapsedString(real, x.PosixFormat))
		// TODO: can we do these?
		r.outf(format, "user", elapsedString(0, x.PosixFormat))
		r.outf(format, "sys", elapsedString(0, x.PosixFormat))
	default:
		panic(fmt.Sprintf("unhandled command node: %T", x))
	}
}

func (r *Runner) flattenAssign(as *syntax.Assign) []*syntax.Assign {
	// Convert "declare $x" into "declare value".
	// Don't use syntax.Parser here, as we only want the basic
	// splitting by '='.
	if as.Name != nil {
		return []*syntax.Assign{as} // nothing to do
	}
	var asgns []*syntax.Assign
	for _, field := range r.fields(as.Value) {
		as := &syntax.Assign{}
		parts := strings.SplitN(field, "=", 2)
		as.Name = &syntax.Lit{Value: parts[0]}
		if len(parts) == 1 {
			as.Naked = true
		} else {
			as.Value = &syntax.Word{Parts: []syntax.WordPart{
				&syntax.Lit{Value: parts[1]},
			}}
		}
		asgns = append(asgns, as)
	}
	return asgns
}

func match(pattern, name string) bool {
	expr, err := syntax.TranslatePattern(pattern, true)
	if err != nil {
		return false
	}
	rx := regexp.MustCompile("^" + expr + "$")
	return rx.MatchString(name)
}

func elapsedString(d time.Duration, posix bool) string {
	if posix {
		return fmt.Sprintf("%.2f", d.Seconds())
	}
	min := int(d.Minutes())
	sec := math.Remainder(d.Seconds(), 60.0)
	return fmt.Sprintf("%dm%.3fs", min, sec)
}

func (r *Runner) stmts(ctx context.Context, sl syntax.StmtList) {
	for _, stmt := range sl.Stmts {
		r.stmt(ctx, stmt)
	}
}

func (r *Runner) hdocReader(rd *syntax.Redirect) io.Reader {
	if rd.Op != syntax.DashHdoc {
		hdoc := r.document(rd.Hdoc)
		return strings.NewReader(hdoc)
	}
	var buf bytes.Buffer
	var cur []syntax.WordPart
	flushLine := func() {
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(r.document(&syntax.Word{Parts: cur}))
		cur = cur[:0]
	}
	for _, wp := range rd.Hdoc.Parts {
		lit, ok := wp.(*syntax.Lit)
		if !ok {
			cur = append(cur, wp)
			continue
		}
		for i, part := range strings.Split(lit.Value, "\n") {
			if i > 0 {
				flushLine()
				cur = cur[:0]
			}
			part = strings.TrimLeft(part, "\t")
			cur = append(cur, &syntax.Lit{Value: part})
		}
	}
	flushLine()
	return &buf
}

func (r *Runner) redir(ctx context.Context, rd *syntax.Redirect) (io.Closer, error) {
	if rd.Hdoc != nil {
		r.Stdin = r.hdocReader(rd)
		return nil, nil
	}
	orig := &r.Stdout
	if rd.N != nil {
		switch rd.N.Value {
		case "1":
		case "2":
			orig = &r.Stderr
		}
	}
	arg := r.literal(rd.Word)
	switch rd.Op {
	case syntax.WordHdoc:
		r.Stdin = strings.NewReader(arg + "\n")
		return nil, nil
	case syntax.DplOut:
		switch arg {
		case "1":
			*orig = r.Stdout
		case "2":
			*orig = r.Stderr
		}
		return nil, nil
	case syntax.RdrIn, syntax.RdrOut, syntax.AppOut,
		syntax.RdrAll, syntax.AppAll:
		// done further below
	// case syntax.DplIn:
	default:
		panic(fmt.Sprintf("unhandled redirect op: %v", rd.Op))
	}
	mode := os.O_RDONLY
	switch rd.Op {
	case syntax.AppOut, syntax.AppAll:
		mode = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case syntax.RdrOut, syntax.RdrAll:
		mode = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := r.open(ctx, r.relPath(arg), mode, 0644, true)
	if err != nil {
		return nil, err
	}
	switch rd.Op {
	case syntax.RdrIn:
		r.Stdin = f
	case syntax.RdrOut, syntax.AppOut:
		*orig = f
	case syntax.RdrAll, syntax.AppAll:
		r.Stdout = f
		r.Stderr = f
	default:
		panic(fmt.Sprintf("unhandled redirect op: %v", rd.Op))
	}
	return f, nil
}

func (r *Runner) loopStmtsBroken(ctx context.Context, sl syntax.StmtList) bool {
	oldInLoop := r.inLoop
	r.inLoop = true
	defer func() { r.inLoop = oldInLoop }()
	for _, stmt := range sl.Stmts {
		r.stmt(ctx, stmt)
		if r.contnEnclosing > 0 {
			r.contnEnclosing--
			return r.contnEnclosing > 0
		}
		if r.breakEnclosing > 0 {
			r.breakEnclosing--
			return true
		}
	}
	return false
}

type returnStatus uint8

func (s returnStatus) Error() string { return fmt.Sprintf("return status %d", s) }

func (r *Runner) call(ctx context.Context, pos syntax.Pos, args []string) {
	if r.stop(ctx) {
		return
	}
	name := args[0]
	if body := r.Funcs[name]; body != nil {
		// stack them to support nested func calls
		oldParams := r.Params
		r.Params = args[1:]
		oldInFunc := r.inFunc
		oldFuncVars := r.funcVars
		r.funcVars = nil
		r.inFunc = true

		r.stmt(ctx, body)

		r.Params = oldParams
		r.funcVars = oldFuncVars
		r.inFunc = oldInFunc
		if code, ok := r.err.(returnStatus); ok {
			r.err = nil
			r.exit = int(code)
		}
		return
	}
	if isBuiltin(name) {
		r.exit = r.builtinCode(ctx, pos, name, args[1:])
		return
	}
	r.exec(ctx, args)
}

func (r *Runner) exec(ctx context.Context, args []string) {
	path := r.lookPath(args[0])
	err := r.Exec(r.modCtx(ctx), path, args)
	switch x := err.(type) {
	case nil:
		r.exit = 0
	case ExitStatus:
		r.exit = int(x)
	default: // module's custom fatal error
		r.setErr(err)
	}
}

func (r *Runner) open(ctx context.Context, path string, flags int, mode os.FileMode, print bool) (io.ReadWriteCloser, error) {
	f, err := r.Open(r.modCtx(ctx), path, flags, mode)
	switch err.(type) {
	case nil:
	case *os.PathError:
		if print {
			r.errf("%v\n", err)
		}
	default: // module's custom fatal error
		r.setErr(err)
	}
	return f, err
}

func (r *Runner) stat(name string) (os.FileInfo, error) {
	return os.Stat(r.relPath(name))
}

func (r *Runner) checkStat(file string) string {
	d, err := r.stat(file)
	if err != nil {
		return ""
	}
	m := d.Mode()
	if m.IsDir() {
		return ""
	}
	if runtime.GOOS != "windows" && m&0111 == 0 {
		return ""
	}
	return file
}

func winHasExt(file string) bool {
	i := strings.LastIndex(file, ".")
	if i < 0 {
		return false
	}
	return strings.LastIndexAny(file, `:\/`) < i
}

func (r *Runner) findExecutable(file string, exts []string) string {
	if len(exts) == 0 {
		// non-windows
		return r.checkStat(file)
	}
	if winHasExt(file) && r.checkStat(file) != "" {
		return file
	}
	for _, e := range exts {
		if f := file + e; r.checkStat(f) != "" {
			return f
		}
	}
	return ""
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

func (r *Runner) lookPath(file string) string {
	pathList := splitList(r.envGet("PATH"))
	chars := `/`
	if runtime.GOOS == "windows" {
		chars = `:\/`
		// so that "foo" always tries "./foo"
		pathList = append([]string{"."}, pathList...)
	}
	exts := r.pathExts()
	if strings.ContainsAny(file, chars) {
		return r.findExecutable(file, exts)
	}
	for _, dir := range pathList {
		var path string
		switch dir {
		case "", ".":
			// otherwise "foo" won't be "./foo"
			path = "." + string(filepath.Separator) + file
		default:
			path = filepath.Join(dir, file)
		}
		if f := r.findExecutable(path, exts); f != "" {
			return f
		}
	}
	return ""
}

func (r *Runner) pathExts() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	pathext := r.envGet("PATHEXT")
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
