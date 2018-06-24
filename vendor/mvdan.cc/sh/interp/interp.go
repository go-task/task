// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"mvdan.cc/sh/syntax"
)

// A Runner interprets shell programs. It cannot be reused once a
// program has been interpreted.
//
// Note that writes to Stdout and Stderr may not be sequential. If
// you plan on using an io.Writer implementation that isn't safe for
// concurrent use, consider a workaround like hiding writes behind a
// mutex.
type Runner struct {
	// Env specifies the environment of the interpreter.
	// If Env is nil, Run uses the current process's environment.
	Env Environ

	// Dir specifies the working directory of the command. If Dir is
	// the empty string, Run runs the command in the calling
	// process's current directory.
	Dir string

	// Params are the current parameters, e.g. from running a shell
	// file or calling a function. Accessible via the $@/$* family
	// of vars.
	Params []string

	Exec ModuleExec
	Open ModuleOpen

	filename string // only if Node was a File

	// Separate maps, note that bash allows a name to be both a var
	// and a func simultaneously
	Vars  map[string]Variable
	Funcs map[string]*syntax.Stmt

	// like Vars, but local to a func i.e. "local foo=bar"
	funcVars map[string]Variable

	// like Vars, but local to a cmd i.e. "foo=bar prog args..."
	cmdVars map[string]string

	// >0 to break or continue out of N enclosing loops
	breakEnclosing, contnEnclosing int

	inLoop   bool
	inFunc   bool
	inSource bool

	err  error // current fatal error
	exit int   // current (last) exit code

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	bgShells sync.WaitGroup

	// Context can be used to cancel the interpreter before it finishes
	Context context.Context

	opts [len(shellOptsTable) + len(bashOptsTable)]bool

	dirStack []string

	optState getopts

	ifsJoin string
	ifsRune func(rune) bool

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

	fieldAlloc  [4]fieldPart
	fieldsAlloc [4][]fieldPart
	bufferAlloc bytes.Buffer
	oneWord     [1]*syntax.Word
}

func (r *Runner) strBuilder() *bytes.Buffer {
	b := &r.bufferAlloc
	b.Reset()
	return b
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

// Reset will set the unexported fields back to zero, fill any exported
// fields with their default values if not set, and prepare the runner
// to interpret a program.
//
// This function should be called once before running any node. It can
// be skipped before any following runs to keep internal state, such as
// declared variables.
func (r *Runner) Reset() error {
	// reset the internal state
	*r = Runner{
		Env:         r.Env,
		Dir:         r.Dir,
		Params:      r.Params,
		Context:     r.Context,
		Stdin:       r.Stdin,
		Stdout:      r.Stdout,
		Stderr:      r.Stderr,
		Exec:        r.Exec,
		Open:        r.Open,
		KillTimeout: r.KillTimeout,

		// emptied below, to reuse the space
		Vars:     r.Vars,
		cmdVars:  r.cmdVars,
		dirStack: r.dirStack[:0],
	}
	if r.Vars == nil {
		r.Vars = make(map[string]Variable)
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
	if r.Context == nil {
		r.Context = context.Background()
	}
	if r.Env == nil {
		r.Env, _ = EnvFromList(os.Environ())
	}
	if _, ok := r.Env.Get("HOME"); !ok {
		u, _ := user.Current()
		r.Vars["HOME"] = Variable{Value: StringVal(u.HomeDir)}
	}
	if r.Dir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get current dir: %v", err)
		}
		r.Dir = dir
	} else {
		abs, err := filepath.Abs(r.Dir)
		if err != nil {
			return fmt.Errorf("could not get absolute dir: %v", err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return fmt.Errorf("could not stat: %v", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", abs)
		}
		r.Dir = abs
	}
	r.Vars["PWD"] = Variable{Value: StringVal(r.Dir)}
	r.Vars["IFS"] = Variable{Value: StringVal(" \t\n")}
	r.ifsUpdated()
	r.Vars["OPTIND"] = Variable{Value: StringVal("1")}

	if runtime.GOOS == "windows" {
		// convert $PATH to a unix path list
		path, _ := r.Env.Get("PATH")
		path = strings.Join(filepath.SplitList(path), ":")
		r.Vars["PATH"] = Variable{Value: StringVal(path)}
	}

	r.dirStack = append(r.dirStack, r.Dir)
	if r.Exec == nil {
		r.Exec = DefaultExec
	}
	if r.Open == nil {
		r.Open = DefaultOpen
	}
	if r.KillTimeout == 0 {
		r.KillTimeout = 2 * time.Second
	}
	return nil
}

func (r *Runner) ctx() Ctxt {
	c := Ctxt{
		Context:     r.Context,
		Env:         r.Env,
		Dir:         r.Dir,
		Stdin:       r.Stdin,
		Stdout:      r.Stdout,
		Stderr:      r.Stderr,
		KillTimeout: r.KillTimeout,
	}
	c.Env = r.Env.Copy()
	for name, vr := range r.Vars {
		if !vr.Exported {
			continue
		}
		c.Env.Set(name, r.varStr(vr, 0))
	}
	for name, val := range r.cmdVars {
		c.Env.Set(name, val)
	}
	return c
}

type ExitCode uint8

func (e ExitCode) Error() string { return fmt.Sprintf("exit status %d", e) }

func (r *Runner) setErr(err error) {
	if r.err == nil {
		r.err = err
	}
}

func (r *Runner) lastExit() {
	if r.err == nil {
		r.err = ExitCode(r.exit)
	}
}

// FromArgs populates the shell options and returns the remaining
// arguments. For example, running FromArgs("-e", "--", "foo") will set
// the "-e" option and return []string{"foo"}.
//
// This is similar to what the interpreter's "set" builtin does.
func (r *Runner) FromArgs(args ...string) ([]string, error) {
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
			return nil, fmt.Errorf("invalid option: %q", arg)
		}
		*opt = enable
		args = args[1:]
	}
	return args, nil
}

// Run starts the interpreter and returns any error.
func (r *Runner) Run(node syntax.Node) error {
	r.filename = ""
	switch x := node.(type) {
	case *syntax.File:
		r.filename = x.Name
		r.stmts(x.StmtList)
	case *syntax.Stmt:
		r.stmt(x)
	case syntax.Command:
		r.cmd(x)
	default:
		return fmt.Errorf("Node can only be File, Stmt, or Command: %T", x)
	}
	r.lastExit()
	if r.err == ExitCode(0) {
		r.err = nil
	}
	return r.err
}

func (r *Runner) Stmt(stmt *syntax.Stmt) error {
	r.stmt(stmt)
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

func (r *Runner) stop() bool {
	if r.err != nil {
		return true
	}
	if err := r.Context.Err(); err != nil {
		r.err = err
		return true
	}
	if r.opts[optNoExec] {
		return true
	}
	return false
}

func (r *Runner) stmt(st *syntax.Stmt) {
	if r.stop() {
		return
	}
	if st.Background {
		r.bgShells.Add(1)
		r2 := r.sub()
		go func() {
			r2.stmtSync(st)
			r.bgShells.Done()
		}()
	} else {
		r.stmtSync(st)
	}
}

func (r *Runner) stmtSync(st *syntax.Stmt) {
	oldIn, oldOut, oldErr := r.Stdin, r.Stdout, r.Stderr
	for _, rd := range st.Redirs {
		cls, err := r.redir(rd)
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
		r.cmd(st.Cmd)
	}
	if st.Negated {
		r.exit = oneIf(r.exit == 0)
	}
	if r.exit != 0 && r.opts[optErrExit] {
		r.lastExit()
	}
	if !r.keepRedirs {
		r.Stdin, r.Stdout, r.Stderr = oldIn, oldOut, oldErr
	}
}

func (r *Runner) sub() *Runner {
	r2 := *r
	r2.bgShells = sync.WaitGroup{}
	r2.bufferAlloc = bytes.Buffer{}
	// TODO: perhaps we could do a lazy copy here, or some sort of
	// overlay to avoid copying all the time
	r2.Env = r.Env.Copy()
	r2.Vars = make(map[string]Variable, len(r.Vars))
	for k, v := range r.Vars {
		r2.Vars[k] = v
	}
	r2.cmdVars = make(map[string]string, len(r.cmdVars))
	for k, v := range r.cmdVars {
		r2.cmdVars[k] = v
	}
	return &r2
}

func (r *Runner) cmd(cm syntax.Command) {
	if r.stop() {
		return
	}
	switch x := cm.(type) {
	case *syntax.Block:
		r.stmts(x.StmtList)
	case *syntax.Subshell:
		r2 := r.sub()
		r2.stmts(x.StmtList)
		r.exit = r2.exit
		r.setErr(r2.err)
	case *syntax.CallExpr:
		fields := r.Fields(x.Args...)
		if len(fields) == 0 {
			for _, as := range x.Assigns {
				vr, _ := r.lookupVar(as.Name.Value)
				vr.Value = r.assignVal(as, "")
				r.setVar(as.Name.Value, as.Index, vr)
			}
			break
		}
		for _, as := range x.Assigns {
			val := r.assignVal(as, "")
			// we know that inline vars must be strings
			r.cmdVars[as.Name.Value] = string(val.(StringVal))
			if as.Name.Value == "IFS" {
				r.ifsUpdated()
				defer r.ifsUpdated()
			}
		}
		r.call(x.Args[0].Pos(), fields)
		// cmdVars can be nuked here, as they are never useful
		// again once we nest into further levels of inline
		// vars.
		for k := range r.cmdVars {
			delete(r.cmdVars, k)
		}
	case *syntax.BinaryCmd:
		switch x.Op {
		case syntax.AndStmt:
			r.stmt(x.X)
			if r.exit == 0 {
				r.stmt(x.Y)
			}
		case syntax.OrStmt:
			r.stmt(x.X)
			if r.exit != 0 {
				r.stmt(x.Y)
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
				r2.stmt(x.X)
				pw.Close()
				wg.Done()
			}()
			r.stmt(x.Y)
			pr.Close()
			wg.Wait()
			if r.opts[optPipeFail] && r2.exit > 0 && r.exit == 0 {
				r.exit = r2.exit
			}
			r.setErr(r2.err)
		}
	case *syntax.IfClause:
		r.stmts(x.Cond)
		if r.exit == 0 {
			r.stmts(x.Then)
			break
		}
		r.exit = 0
		r.stmts(x.Else)
	case *syntax.WhileClause:
		for !r.stop() {
			r.stmts(x.Cond)
			stop := (r.exit == 0) == x.Until
			r.exit = 0
			if stop || r.loopStmtsBroken(x.Do) {
				break
			}
		}
	case *syntax.ForClause:
		switch y := x.Loop.(type) {
		case *syntax.WordIter:
			name := y.Name.Value
			for _, field := range r.Fields(y.Items...) {
				r.setVarString(name, field)
				if r.loopStmtsBroken(x.Do) {
					break
				}
			}
		case *syntax.CStyleLoop:
			r.arithm(y.Init)
			for r.arithm(y.Cond) != 0 {
				if r.loopStmtsBroken(x.Do) {
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
		str := r.loneWord(x.Word)
		for _, ci := range x.Items {
			for _, word := range ci.Patterns {
				pat := r.lonePattern(word)
				if match(pat, str) {
					r.stmts(ci.StmtList)
					return
				}
			}
		}
	case *syntax.TestClause:
		r.exit = 0
		if r.bashTest(x.X) == "" && r.exit == 0 {
			// to preserve exit code 2 for regex
			// errors, etc
			r.exit = 1
		}
	case *syntax.DeclClause:
		local := false
		var modes []string
		valType := ""
		switch x.Variant.Value {
		case "declare":
			// When used in a function, "declare" acts as
			// "local" unless the "-g" option is used.
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
			switch s := r.loneWord(opt); s {
			case "-x", "-r", "-n":
				modes = append(modes, s)
			case "-a", "-A":
				valType = s
			case "-g":
				local = false
			default:
				r.errf("declare: invalid option %q\n", s)
				r.exit = 2
				return
			}
		}
		for _, as := range x.Assigns {
			for _, as := range r.expandAssigns(as) {
				name := as.Name.Value
				vr, _ := r.lookupVar(as.Name.Value)
				vr.Value = r.assignVal(as, valType)
				vr.Local = local
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
			r.stmt(x.Stmt)
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

func elapsedString(d time.Duration, posix bool) string {
	if posix {
		return fmt.Sprintf("%.2f", d.Seconds())
	}
	min := int(d.Minutes())
	sec := math.Remainder(d.Seconds(), 60.0)
	return fmt.Sprintf("%dm%.3fs", min, sec)
}

func (r *Runner) stmts(sl syntax.StmtList) {
	for _, stmt := range sl.Stmts {
		r.stmt(stmt)
	}
}

func (r *Runner) redir(rd *syntax.Redirect) (io.Closer, error) {
	if rd.Hdoc != nil {
		hdoc := r.loneWord(rd.Hdoc)
		r.Stdin = strings.NewReader(hdoc)
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
	arg := r.loneWord(rd.Word)
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
		mode = os.O_RDWR | os.O_CREATE | os.O_APPEND
	case syntax.RdrOut, syntax.RdrAll:
		mode = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	}
	f, err := r.open(r.relPath(arg), mode, 0644, true)
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

func (r *Runner) loopStmtsBroken(sl syntax.StmtList) bool {
	oldInLoop := r.inLoop
	r.inLoop = true
	defer func() { r.inLoop = oldInLoop }()
	for _, stmt := range sl.Stmts {
		r.stmt(stmt)
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

type returnCode uint8

func (returnCode) Error() string { return "returned" }

func (r *Runner) call(pos syntax.Pos, args []string) {
	if r.stop() {
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

		r.stmt(body)

		r.Params = oldParams
		r.funcVars = oldFuncVars
		r.inFunc = oldInFunc
		if code, ok := r.err.(returnCode); ok {
			r.err = nil
			r.exit = int(code)
		}
		return
	}
	if isBuiltin(name) {
		r.exit = r.builtinCode(pos, name, args[1:])
		return
	}
	r.exec(args)
}

func (r *Runner) exec(args []string) {
	path := r.lookPath(args[0])
	err := r.Exec(r.ctx(), path, args)
	switch x := err.(type) {
	case nil:
		r.exit = 0
	case ExitCode:
		r.exit = int(x)
	default: // module's custom fatal error
		r.setErr(err)
	}
}

func (r *Runner) open(path string, flags int, mode os.FileMode, print bool) (io.ReadWriteCloser, error) {
	f, err := r.Open(r.ctx(), path, flags, mode)
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
		case len(s) != 1, s[0] < 'A', s[0] > 'Z':
			// not a disk name
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
	pathList := splitList(r.getVar("PATH"))
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
	pathext := r.getVar("PATHEXT")
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
