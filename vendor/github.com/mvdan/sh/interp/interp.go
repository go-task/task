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
	"path/filepath"
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

func fieldJoin(parts []fieldPart) string {
	var buf bytes.Buffer
	for _, part := range parts {
		buf.WriteString(part.val)
	}
	return buf.String()
}

func escapedGlob(parts []fieldPart) (escaped string, glob bool) {
	var buf bytes.Buffer
	for _, part := range parts {
		for _, r := range part.val {
			switch r {
			case '*', '?', '\\', '[':
				if part.quoted {
					buf.WriteByte('\\')
				} else {
					glob = true
				}
			}
			buf.WriteRune(r)
		}
	}
	return buf.String(), glob
}

func (r *Runner) fields(words []*syntax.Word) []string {
	fields := make([]string, 0, len(words))
	baseDir, _ := escapedGlob([]fieldPart{{val: r.Dir}})
	for _, word := range words {
		for _, field := range r.wordFields(word.Parts, false) {
			path, glob := escapedGlob(field)
			var matches []string
			abs := filepath.IsAbs(path)
			if glob {
				if !abs {
					path = filepath.Join(baseDir, path)
				}
				matches, _ = filepath.Glob(path)
			}
			if len(matches) == 0 {
				fields = append(fields, fieldJoin(field))
				continue
			}
			for _, match := range matches {
				if !abs {
					match, _ = filepath.Rel(baseDir, match)
				}
				fields = append(fields, match)
			}
		}
	}
	return fields
}

func (r *Runner) loneWord(word *syntax.Word) string {
	if word == nil {
		return ""
	}
	var buf bytes.Buffer
	for _, field := range r.wordFields(word.Parts, false) {
		for _, part := range field {
			buf.WriteString(part.val)
		}
	}
	return buf.String()
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

func (r *Runner) assignValue(as *syntax.Assign) varValue {
	prev, _ := r.lookupVar(as.Name.Value)
	if as.Value != nil {
		s := r.loneWord(as.Value)
		if !as.Append || prev == nil {
			return s
		}
		switch x := prev.(type) {
		case string:
			return x + s
		case []string:
			if len(x) == 0 {
				return []string{s}
			}
			x[0] += s
			return x
		}
		return s
	}
	if as.Array != nil {
		strs := make([]string, len(as.Array.Elems))
		for i, elem := range as.Array.Elems {
			strs[i] = r.loneWord(elem.Value)
		}
		if !as.Append || prev == nil {
			return strs
		}
		switch x := prev.(type) {
		case string:
			return append([]string{x}, strs...)
		case []string:
			return append(x, strs...)
		}
		return strs
	}
	return nil
}

func (r *Runner) stmtSync(st *syntax.Stmt) {
	oldVars := r.cmdVars
	for _, as := range st.Assigns {
		val := r.assignValue(as)
		if st.Cmd == nil {
			r.setVar(as.Name.Value, val)
			continue
		}
		if r.cmdVars == nil {
			r.cmdVars = make(map[string]varValue, len(st.Assigns))
		}
		r.cmdVars[as.Name.Value] = val
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
		r.exit = 0
		for _, el := range x.Elifs {
			r.stmts(el.CondStmts)
			if r.exit == 0 {
				r.stmts(el.ThenStmts)
				return
			}
		}
		r.stmts(x.ElseStmts)
	case *syntax.WhileClause:
		for r.err == nil {
			r.stmts(x.CondStmts)
			stop := (r.exit == 0) == x.Until
			r.exit = 0
			if stop || r.loopStmtsBroken(x.DoStmts) {
				break
			}
		}
	case *syntax.ForClause:
		switch y := x.Loop.(type) {
		case *syntax.WordIter:
			name := y.Name.Value
			for _, field := range r.fields(y.Items) {
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
		for _, ci := range x.Items {
			for _, word := range ci.Patterns {
				var buf bytes.Buffer
				for _, field := range r.wordFields(word.Parts, false) {
					escaped, _ := escapedGlob(field)
					buf.WriteString(escaped)
				}
				if match(buf.String(), str) {
					r.stmts(ci.Stmts)
					return
				}
			}
		}
	case *syntax.TestClause:
		if r.bashTest(x.X) == "" && r.exit == 0 {
			r.exit = 1
		}
	case *syntax.DeclClause:
		if len(x.Opts) > 0 {
			r.runErr(cm.Pos(), "unhandled declare opts")
		}
		for _, as := range x.Assigns {
			r.setVar(as.Name.Value, r.assignValue(as))
		}
	default:
		r.runErr(cm.Pos(), "unhandled command node: %T", x)
	}
}

func (r *Runner) stmts(stmts []*syntax.Stmt) {
	for _, stmt := range stmts {
		r.stmt(stmt)
	}
}

func match(pattern, name string) bool {
	matched, _ := path.Match(pattern, name)
	return matched
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
		r.runErr(rd.Pos(), "unhandled redirect op: %v", rd.Op)
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
		r.runErr(rd.Pos(), "unhandled redirect op: %v", rd.Op)
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

type fieldPart struct {
	val    string
	quoted bool
}

func (r *Runner) wordFields(wps []syntax.WordPart, quoted bool) [][]fieldPart {
	var fields [][]fieldPart
	var curField []fieldPart
	allowEmpty := false
	flush := func() {
		if len(curField) == 0 {
			return
		}
		fields = append(fields, curField)
		curField = nil
	}
	splitAdd := func(val string) {
		// TODO: use IFS
		for i, field := range strings.Fields(val) {
			if i > 0 {
				flush()
			}
			curField = append(curField, fieldPart{val: field})
		}
	}
	for i, wp := range wps {
		switch x := wp.(type) {
		case *syntax.Lit:
			s := x.Value
			if i > 0 || len(s) == 0 || s[0] != '~' {
			} else if len(s) < 2 || s[1] == '/' {
				// TODO: ~someuser
				s = r.getVar("HOME") + s[1:]
			}
			curField = append(curField, fieldPart{val: s})
		case *syntax.SglQuoted:
			allowEmpty = true
			curField = append(curField, fieldPart{
				quoted: true,
				val:    x.Value,
			})
		case *syntax.DblQuoted:
			allowEmpty = true
			if len(x.Parts) == 1 {
				pe, _ := x.Parts[0].(*syntax.ParamExp)
				if elems := r.quotedElems(pe); elems != nil {
					for i, elem := range elems {
						if i > 0 {
							flush()
						}
						curField = append(curField, fieldPart{
							quoted: true,
							val:    elem,
						})
					}
					continue
				}
			}
			for _, field := range r.wordFields(x.Parts, true) {
				for _, part := range field {
					curField = append(curField, fieldPart{
						quoted: true,
						val:    part.val,
					})
				}
			}
		case *syntax.ParamExp:
			val := r.paramExp(x)
			if quoted {
				curField = append(curField, fieldPart{val: val})
			} else {
				splitAdd(val)
			}
		case *syntax.CmdSubst:
			r2 := *r
			var buf bytes.Buffer
			r2.Stdout = &buf
			r2.stmts(x.Stmts)
			val := strings.TrimRight(buf.String(), "\n")
			if quoted {
				curField = append(curField, fieldPart{val: val})
			} else {
				splitAdd(val)
			}
		case *syntax.ArithmExp:
			curField = append(curField, fieldPart{
				val: strconv.Itoa(r.arithm(x.X)),
			})
		default:
			r.runErr(wp.Pos(), "unhandled word part: %T", x)
		}
	}
	flush()
	if allowEmpty && len(fields) == 0 {
		fields = append(fields, []fieldPart{{}})
	}
	return fields
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
	if isBuiltin(name) {
		r.exit = r.builtinCode(pos, name, args)
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
		r.exit = 1
		if status, ok := x.Sys().(syscall.WaitStatus); ok {
			r.exit = status.ExitStatus()
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
