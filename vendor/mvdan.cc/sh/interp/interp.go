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
	Env []string

	// envMap is just Env as a map, to simplify and speed up its use
	envMap map[string]string

	// Dir specifies the working directory of the command. If Dir is
	// the empty string, Run runs the command in the calling
	// process's current directory.
	Dir string

	// Params are the current parameters, e.g. from running a shell
	// file or calling a function. Accessible via the $@/$* family
	// of vars.
	Params []string

	filename string // only if Node was a File

	// Separate maps, note that bash allows a name to be both a var
	// and a func simultaneously
	vars  map[string]varValue
	funcs map[string]*syntax.Stmt

	// like vars, but local to a cmd i.e. "foo=bar prog args..."
	cmdVars map[string]varValue

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

	stopOnCmdErr bool // set -e
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
	Filename string
	syntax.Pos
	Text string
}

func (e RunError) Error() string {
	if e.Filename == "" {
		return fmt.Sprintf("%s: %s", e.Pos.String(), e.Text)
	}
	return fmt.Sprintf("%s:%s: %s", e.Filename, e.Pos.String(), e.Text)
}

func (r *Runner) runErr(pos syntax.Pos, format string, a ...interface{}) {
	if r.err == nil {
		r.err = RunError{
			Filename: r.filename,
			Pos:      pos,
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

// FromArgs populates the shell options and returns the remaining
// arguments. For example, running FromArgs("-e", "--", "foo") will set
// the "-e" option and return []string{"foo"}.
//
// This is similar to what the interpreter's "set" builtin does.
func (r *Runner) FromArgs(args ...string) ([]string, error) {
opts:
	for len(args) > 0 {
		opt := args[0]
		if opt == "" || (opt[0] != '-' && opt[0] != '+') {
			break
		}
		enable := opt[0] == '-'
		switch opt[1:] {
		case "-":
			args = args[1:]
			break opts
		case "e":
			r.stopOnCmdErr = enable
		default:
			return nil, fmt.Errorf("invalid option: %q", opt)
		}
		args = args[1:]
	}
	return args, nil
}

func (r *Runner) Reset() error {
	// reset the internal state
	*r = Runner{
		Env:     r.Env,
		Dir:     r.Dir,
		Params:  r.Params,
		Context: r.Context,
		Stdin:   r.Stdin,
		Stdout:  r.Stdout,
		Stderr:  r.Stderr,
	}
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
	return nil
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

func (r *Runner) outf(format string, a ...interface{}) {
	fmt.Fprintf(r.Stdout, format, a...)
}

func (r *Runner) errf(format string, a ...interface{}) {
	fmt.Fprintf(r.Stderr, format, a...)
}

func (r *Runner) expand(format string, onlyChars bool, args ...string) string {
	var buf bytes.Buffer
	esc, fmt := false, false
	for _, c := range format {
		if esc {
			esc = false
			switch c {
			case 'n':
				buf.WriteRune('\n')
			case 'r':
				buf.WriteRune('\r')
			case 't':
				buf.WriteRune('\t')
			case '\\':
				buf.WriteRune('\\')
			default:
				buf.WriteRune('\\')
				buf.WriteRune(c)
			}
			continue
		}
		if fmt {
			fmt = false
			arg := ""
			n := 0
			if len(args) > 0 {
				arg, args = args[0], args[1:]
				i, _ := strconv.ParseInt(arg, 0, 0)
				n = int(i)
			}
			switch c {
			case 's':
				buf.WriteString(arg)
			case 'c':
				var b byte
				if len(arg) > 0 {
					b = arg[0]
				}
				buf.WriteByte(b)
			case 'd', 'i':
				buf.WriteString(strconv.Itoa(n))
			case 'u':
				buf.WriteString(strconv.FormatUint(uint64(n), 10))
			case 'o':
				buf.WriteString(strconv.FormatUint(uint64(n), 8))
			case 'x':
				buf.WriteString(strconv.FormatUint(uint64(n), 16))
			default:
				r.runErr(syntax.Pos{}, "unhandled format char: %c", c)
			}
			continue
		}
		if c == '\\' {
			esc = true
		} else if !onlyChars && c == '%' {
			fmt = true
		} else {
			buf.WriteRune(c)
		}
	}
	return buf.String()
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
		r.stmts(x.StmtList)
	case *syntax.Subshell:
		r2 := *r
		r2.stmts(x.StmtList)
		r.exit = r2.exit
	case *syntax.CallExpr:
		if len(x.Args) == 0 {
			for _, as := range x.Assigns {
				r.setVar(as.Name.Value, r.assignValue(as))
			}
			break
		}
		oldVars := r.cmdVars
		if r.cmdVars == nil {
			r.cmdVars = make(map[string]varValue, len(x.Assigns))
		}
		for _, as := range x.Assigns {
			r.cmdVars[as.Name.Value] = r.assignValue(as)
		}
		fields := r.fields(x.Args)
		r.call(x.Args[0].Pos(), fields[0], fields[1:])
		r.cmdVars = oldVars
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
		r.stmts(x.Cond)
		if r.exit == 0 {
			r.stmts(x.Then)
			return
		}
		r.exit = 0
		r.stmts(x.Else)
	case *syntax.WhileClause:
		for r.err == nil {
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
			for _, field := range r.fields(y.Items) {
				r.setVar(name, field)
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
					r.stmts(ci.StmtList)
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
	if r.exit != 0 && r.stopOnCmdErr {
		r.lastExit()
	}
}

func (r *Runner) stmts(sl syntax.StmtList) {
	for _, stmt := range sl.Stmts {
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
	var f io.ReadWriteCloser
	switch arg {
	case "/dev/null":
		f = devNull{}
	default:
		var err error
		f, err = os.OpenFile(r.relPath(arg), mode, 0644)
		if err != nil {
			// TODO: print to stderr?
			return nil, err
		}
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

func (r *Runner) loopStmtsBroken(sl syntax.StmtList) bool {
	r.inLoop = true
	defer func() { r.inLoop = false }()
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
			fp := fieldPart{quoted: true, val: x.Value}
			if x.Dollar {
				fp.val = r.expand(fp.val, true)
			}
			curField = append(curField, fp)
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
			r2.stmts(x.StmtList)
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
		oldParams := r.Params
		r.Params = args
		r.stmt(body)
		r.Params = oldParams
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
