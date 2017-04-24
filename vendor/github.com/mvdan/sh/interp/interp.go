// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/mvdan/sh/syntax"
)

// A Runner interprets shell programs. It cannot be reused once a
// program has been interpreted.
//
// Note that writes to Stdout and Stderr may not be sequential. If
// you plan on using an io.Writer implementation that isn't safe for
// concurrent use, consider a workaround like hiding writes behind a
// mutex.
type Runner struct {
	// TODO: syntax.Node instead of *syntax.File?
	File *syntax.File

	// Env specifies the environment of the interpreter.
	// If Env is nil, Run uses the current process's environment.
	Env []string

	// envMap is just Env as a map, to simplify and speed up its use
	envMap map[string]string

	// Dir specifies the working directory of the command. If Dir is
	// the empty string, Run runs the command in the calling
	// process's current directory.
	Dir string

	// Separate maps, note that bash allows a name to be both a var
	// and a func simultaneously
	vars  map[string]varValue
	funcs map[string]*syntax.Stmt

	// like vars, but local to a cmd i.e. "foo=bar prog args..."
	cmdVars map[string]varValue

	// Current arguments, if executing a function
	args []string

	// >0 to break or continue out of N enclosing loops
	breakEnclosing, contnEnclosing int

	inLoop bool

	err  error // current fatal error
	exit int   // current (last) exit code

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	bgShells sync.WaitGroup

	// Context can be used to cancel the interpreter before it finishes
	Context context.Context
}

// varValue can hold a string, an indexed array ([]string) or an
// associative array (map[string]string)
// TODO: implement associative arrays
type varValue interface{}

func varStr(v varValue) string {
	switch x := v.(type) {
	case string:
		return x
	case []string:
		if len(x) > 0 {
			return x[0]
		}
	}
	return ""
}

func (r *Runner) varInd(v varValue, e syntax.ArithmExpr) string {
	switch x := v.(type) {
	case string:
		i := r.arithm(e)
		if i == 0 {
			return x
		}
	case []string:
		// TODO: @ between double quotes
		if w, ok := e.(*syntax.Word); ok {
			if lit, ok := w.Parts[0].(*syntax.Lit); ok {
				switch lit.Value {
				case "@", "*":
					return strings.Join(x, " ")
				}
			}
		}
		i := r.arithm(e)
		if len(x) > 0 {
			return x[i]
		}
	}
	return ""
}

type ExitCode uint8

func (e ExitCode) Error() string { return fmt.Sprintf("exit status %d", e) }

type RunError struct {
	syntax.Position
	Text string
}

func (e RunError) Error() string {
	return fmt.Sprintf("%s: %s", e.Position.String(), e.Text)
}

func (r *Runner) runErr(pos syntax.Pos, format string, a ...interface{}) {
	if r.err == nil {
		r.err = RunError{
			Position: r.File.Position(pos),
			Text:     fmt.Sprintf(format, a...),
		}
	}
}

func (r *Runner) lastExit() {
	if r.err == nil {
		r.err = ExitCode(r.exit)
	}
}

func (r *Runner) setVar(name string, val varValue) {
	if r.vars == nil {
		r.vars = make(map[string]varValue, 4)
	}
	r.vars[name] = val
}

func (r *Runner) lookupVar(name string) (varValue, bool) {
	switch name {
	case "PWD":
		return r.Dir, true
	case "HOME":
		u, _ := user.Current()
		return u.HomeDir, true
	}
	if val, e := r.cmdVars[name]; e {
		return val, true
	}
	if val, e := r.vars[name]; e {
		return val, true
	}
	str, e := r.envMap[name]
	return str, e
}

func (r *Runner) getVar(name string) string {
	val, _ := r.lookupVar(name)
	return varStr(val)
}

func (r *Runner) delVar(name string) {
	delete(r.vars, name)
	delete(r.envMap, name)
}

func (r *Runner) setFunc(name string, body *syntax.Stmt) {
	if r.funcs == nil {
		r.funcs = make(map[string]*syntax.Stmt, 4)
	}
	r.funcs[name] = body
}

// Run starts the interpreter and returns any error.
func (r *Runner) Run() error {
	if r.Context == nil {
		r.Context = context.Background()
	}
	if r.Env == nil {
		r.Env = os.Environ()
	}
	r.envMap = make(map[string]string, len(r.Env))
	for _, kv := range r.Env {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			return fmt.Errorf("env not in the form key=value: %q", kv)
		}
		name, val := kv[:i], kv[i+1:]
		r.envMap[name] = val
	}
	if r.Dir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get current dir: %v", err)
		}
		r.Dir = dir
	}
	r.stmts(r.File.Stmts)
	r.lastExit()
	if r.err == ExitCode(0) {
		r.err = nil
	}
	return r.err
}

func (r *Runner) outf(format string, a ...interface{}) {
	fmt.Fprintf(r.Stdout, format, a...)
}

func (r *Runner) errf(format string, a ...interface{}) {
	fmt.Fprintf(r.Stderr, format, a...)
}

func (r *Runner) fields(words []*syntax.Word) []string {
	fields := make([]string, 0, len(words))
	for _, word := range words {
		fields = append(fields, r.wordParts(word.Parts, false)...)
	}
	return fields
}

func (r *Runner) loneWord(word *syntax.Word) string {
	return strings.Join(r.wordParts(word.Parts, false), "")
}

func (r *Runner) stop() bool {
	if r.err != nil {
		return true
	}
	if err := r.Context.Err(); err != nil {
		r.err = err
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
		r2 := *r
		r2.bgShells = sync.WaitGroup{}
		go func() {
			r2.stmtSync(st)
			r.bgShells.Done()
		}()
	} else {
		r.stmtSync(st)
	}
}

func (r *Runner) assignValue(word *syntax.Word) varValue {
	if word == nil {
		return nil
	}
	ae, ok := word.Parts[0].(*syntax.ArrayExpr)
	if !ok {
		return r.loneWord(word)
	}
	strs := make([]string, len(ae.List))
	for i, w := range ae.List {
		strs[i] = r.loneWord(w)
	}
	return strs
}

func (r *Runner) stmtSync(st *syntax.Stmt) {
	oldVars := r.cmdVars
	for _, as := range st.Assigns {
		name := as.Name.Value
		val := r.assignValue(as.Value)
		if st.Cmd == nil {
			r.setVar(name, val)
			continue
		}
		if r.cmdVars == nil {
			r.cmdVars = make(map[string]varValue, len(st.Assigns))
		}
		r.cmdVars[name] = val
	}
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
	r.cmdVars = oldVars
	r.Stdin, r.Stdout, r.Stderr = oldIn, oldOut, oldErr
}

func oneIf(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (r *Runner) cmd(cm syntax.Command) {
	if r.stop() {
		return
	}
	switch x := cm.(type) {
	case *syntax.Block:
		r.stmts(x.Stmts)
	case *syntax.Subshell:
		r2 := *r
		r2.stmts(x.Stmts)
		r.exit = r2.exit
	case *syntax.CallExpr:
		fields := r.fields(x.Args)
		r.call(x.Args[0].Pos(), fields[0], fields[1:])
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
			r2 := *r
			r2.Stdin = r.Stdin
			r2.Stdout = pw
			if x.Op == syntax.PipeAll {
				r2.Stderr = pw
			} else {
				r2.Stderr = r.Stderr
			}
			r.Stdin = pr
			go func() {
				r2.stmt(x.X)
				pw.Close()
			}()
			r.stmt(x.Y)
			pr.Close()
		}
	case *syntax.IfClause:
		r.stmts(x.CondStmts)
		if r.exit == 0 {
			r.stmts(x.ThenStmts)
			return
		}
		for _, el := range x.Elifs {
			r.stmts(el.CondStmts)
			if r.exit == 0 {
				r.stmts(el.ThenStmts)
				return
			}
		}
		r.stmts(x.ElseStmts)
		if len(x.Elifs)+len(x.ElseStmts) == 0 {
			r.exit = 0
		}
	case *syntax.WhileClause:
		for r.err == nil {
			r.stmts(x.CondStmts)
			if r.exit != 0 {
				r.exit = 0
				break
			}
			if r.loopStmtsBroken(x.DoStmts) {
				break
			}
		}
	case *syntax.UntilClause:
		for r.err == nil {
			r.stmts(x.CondStmts)
			if r.exit == 0 {
				break
			}
			r.exit = 0
			if r.loopStmtsBroken(x.DoStmts) {
				break
			}
		}
	case *syntax.ForClause:
		switch y := x.Loop.(type) {
		case *syntax.WordIter:
			name := y.Name.Value
			for _, field := range r.fields(y.List) {
				r.setVar(name, field)
				if r.loopStmtsBroken(x.DoStmts) {
					break
				}
			}
		case *syntax.CStyleLoop:
			r.arithm(y.Init)
			for r.arithm(y.Cond) != 0 {
				if r.loopStmtsBroken(x.DoStmts) {
					break
				}
				r.arithm(y.Post)
			}
		}
	case *syntax.FuncDecl:
		r.setFunc(x.Name.Value, x.Body)
	case *syntax.ArithmCmd:
		if r.arithm(x.X) == 0 {
			r.exit = 1
		}
	case *syntax.LetClause:
		var val int
		for _, expr := range x.Exprs {
			val = r.arithm(expr)
		}
		if val == 0 {
			r.exit = 1
		}
	case *syntax.CaseClause:
		str := r.loneWord(x.Word)
		for _, pl := range x.List {
			for _, word := range pl.Patterns {
				pat := r.loneWord(word)
				// TODO: error?
				matched, _ := path.Match(pat, str)
				if matched {
					r.stmts(pl.Stmts)
					return
				}
			}
		}
	case *syntax.TestClause:
		if r.bashTest(x.X) == "" && r.exit == 0 {
			r.exit = 1
		}
	default:
		r.errf("unhandled command node: %T", x)
	}
}

func (r *Runner) stmts(stmts []*syntax.Stmt) {
	for _, stmt := range stmts {
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
	case syntax.DplIn:
		r.errf("unhandled redirect op: %v", rd.Op)
	}
	mode := os.O_RDONLY
	switch rd.Op {
	case syntax.AppOut, syntax.AppAll:
		mode = os.O_RDWR | os.O_CREATE | os.O_APPEND
	case syntax.RdrOut, syntax.RdrAll:
		mode = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(arg, mode, 0644)
	if err != nil {
		// TODO: print to stderr?
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
		r.errf("unhandled redirect op: %v", rd.Op)
	}
	return f, nil
}

func (r *Runner) loopStmtsBroken(stmts []*syntax.Stmt) bool {
	r.inLoop = true
	defer func() { r.inLoop = false }()
	for _, stmt := range stmts {
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

func (r *Runner) wordParts(wps []syntax.WordPart, quoted bool) []string {
	var parts []string
	var curBuf bytes.Buffer
	flush := func() {
		if curBuf.Len() == 0 {
			return
		}
		parts = append(parts, curBuf.String())
		curBuf.Reset()
	}
	splitAdd := func(val string) {
		// TODO: use IFS
		for i, field := range strings.Fields(val) {
			if i > 0 {
				flush()
			}
			curBuf.WriteString(field)
		}
	}
	for _, wp := range wps {
		switch x := wp.(type) {
		case *syntax.Lit:
			curBuf.WriteString(x.Value)
		case *syntax.SglQuoted:
			curBuf.WriteString(x.Value)
		case *syntax.DblQuoted:
			// TODO: @ between double quotes but not alone
			if len(x.Parts) == 1 {
				pe, ok := x.Parts[0].(*syntax.ParamExp)
				if ok && pe.Param.Value == "@" {
					for i, arg := range r.args {
						if i > 0 {
							flush()
						}
						curBuf.WriteString(arg)
					}
					continue
				}
			}
			for _, str := range r.wordParts(x.Parts, true) {
				curBuf.WriteString(str)
			}
		case *syntax.ParamExp:
			val := r.paramExp(x)
			if quoted {
				curBuf.WriteString(val)
			} else {
				splitAdd(val)
			}
		case *syntax.CmdSubst:
			r2 := *r
			var outBuf bytes.Buffer
			r2.Stdout = &outBuf
			r2.stmts(x.Stmts)
			val := strings.TrimRight(outBuf.String(), "\n")
			if quoted {
				curBuf.WriteString(val)
			} else {
				splitAdd(val)
			}
		case *syntax.ArithmExp:
			curBuf.WriteString(strconv.Itoa(r.arithm(x.X)))
		default:
			r.errf("unhandled word part: %T", x)
		}
	}
	flush()
	return parts
}

func (r *Runner) call(pos syntax.Pos, name string, args []string) {
	if body := r.funcs[name]; body != nil {
		// stack them to support nested func calls
		oldArgs := r.args
		r.args = args
		r.stmt(body)
		r.args = oldArgs
		return
	}
	if r.builtin(pos, name, args) {
		return
	}
	cmd := exec.CommandContext(r.Context, name, args...)
	cmd.Env = r.Env
	for name, val := range r.cmdVars {
		cmd.Env = append(cmd.Env, name+"="+varStr(val))
	}
	cmd.Dir = r.Dir
	cmd.Stdin = r.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	err := cmd.Run()
	switch x := err.(type) {
	case *exec.ExitError:
		// started, but errored - default to 1 if OS
		// doesn't have exit statuses
		if status, ok := x.Sys().(syscall.WaitStatus); ok {
			r.exit = status.ExitStatus()
		} else {
			r.exit = 1
		}
	case *exec.Error:
		// did not start
		// TODO: can this be anything other than
		// "command not found"?
		r.exit = 127
		// TODO: print something?
	default:
		r.exit = 0
	}
}
