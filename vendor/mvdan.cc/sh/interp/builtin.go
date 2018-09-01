// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"mvdan.cc/sh/syntax"
)

func isBuiltin(name string) bool {
	switch name {
	case "true", ":", "false", "exit", "set", "shift", "unset",
		"echo", "printf", "break", "continue", "pwd", "cd",
		"wait", "builtin", "trap", "type", "source", ".", "command",
		"dirs", "pushd", "popd", "umask", "alias", "unalias",
		"fg", "bg", "getopts", "eval", "test", "[", "exec",
		"return", "read", "shopt":
		return true
	}
	return false
}

func (r *Runner) builtinCode(ctx context.Context, pos syntax.Pos, name string, args []string) int {
	switch name {
	case "true", ":":
	case "false":
		return 1
	case "exit":
		switch len(args) {
		case 0:
		case 1:
			if n, err := strconv.Atoi(args[0]); err != nil {
				r.errf("invalid exit status code: %q\n", args[0])
				r.exit = 2
			} else {
				r.exit = n
			}
		default:
			r.errf("exit cannot take multiple arguments\n")
			r.exit = 1
		}
		r.setErr(ShellExitStatus(r.exit))
		return 0 // the command's exit status does not matter
	case "set":
		if err := Params(args...)(r); err != nil {
			r.errf("set: %v\n", err)
			return 2
		}
	case "shift":
		n := 1
		switch len(args) {
		case 0:
		case 1:
			if n2, err := strconv.Atoi(args[0]); err == nil {
				n = n2
				break
			}
			fallthrough
		default:
			r.errf("usage: shift [n]\n")
			return 2
		}
		if n >= len(r.Params) {
			r.Params = nil
		} else {
			r.Params = r.Params[n:]
		}
	case "unset":
		vars := true
		funcs := true
	unsetOpts:
		for i, arg := range args {
			switch arg {
			case "-v":
				funcs = false
			case "-f":
				vars = false
			default:
				args = args[i:]
				break unsetOpts
			}
		}

		for _, arg := range args {
			if _, ok := r.lookupVar(arg); ok && vars {
				r.delVar(arg)
				continue
			}
			if _, ok := r.Funcs[arg]; ok && funcs {
				delete(r.Funcs, arg)
			}
		}
	case "echo":
		newline, expand := true, false
	echoOpts:
		for len(args) > 0 {
			switch args[0] {
			case "-n":
				newline = false
			case "-e":
				expand = true
			case "-E": // default
			default:
				break echoOpts
			}
			args = args[1:]
		}
		for i, arg := range args {
			if i > 0 {
				r.out(" ")
			}
			if expand {
				_, arg, _ = r.expandFormat(arg, nil)
			}
			r.out(arg)
		}
		if newline {
			r.out("\n")
		}
	case "printf":
		if len(args) == 0 {
			r.errf("usage: printf format [arguments]\n")
			return 2
		}
		format, args := args[0], args[1:]
		for {
			n, s, err := r.expandFormat(format, args)
			if err != nil {
				r.errf("%v\n", err)
				return 1
			}
			r.out(s)
			args = args[n:]
			if n == 0 || len(args) == 0 {
				break
			}
		}
	case "break":
		if !r.inLoop {
			r.errf("break is only useful in a loop")
			break
		}
		switch len(args) {
		case 0:
			r.breakEnclosing = 1
		case 1:
			if n, err := strconv.Atoi(args[0]); err == nil {
				r.breakEnclosing = n
				break
			}
			fallthrough
		default:
			r.errf("usage: break [n]\n")
			return 2
		}
	case "continue":
		if !r.inLoop {
			r.errf("continue is only useful in a loop")
			break
		}
		switch len(args) {
		case 0:
			r.contnEnclosing = 1
		case 1:
			if n, err := strconv.Atoi(args[0]); err == nil {
				r.contnEnclosing = n
				break
			}
			fallthrough
		default:
			r.errf("usage: continue [n]\n")
			return 2
		}
	case "pwd":
		r.outf("%s\n", r.getVar("PWD"))
	case "cd":
		var path string
		switch len(args) {
		case 0:
			path = r.getVar("HOME")
		case 1:
			path = args[0]
		default:
			r.errf("usage: cd [dir]\n")
			return 2
		}
		return r.changeDir(path)
	case "wait":
		if len(args) > 0 {
			panic("wait with args not handled yet")
		}
		switch err := r.bgShells.Wait().(type) {
		case nil:
		case ExitStatus:
		case ShellExitStatus:
		default:
			r.setErr(err)
		}
	case "builtin":
		if len(args) < 1 {
			break
		}
		if !isBuiltin(args[0]) {
			return 1
		}
		return r.builtinCode(ctx, pos, args[0], args[1:])
	case "type":
		anyNotFound := false
		for _, arg := range args {
			if _, ok := r.Funcs[arg]; ok {
				r.outf("%s is a function\n", arg)
				continue
			}
			if isBuiltin(arg) {
				r.outf("%s is a shell builtin\n", arg)
				continue
			}
			if path, err := exec.LookPath(arg); err == nil {
				r.outf("%s is %s\n", arg, path)
				continue
			}
			r.errf("type: %s: not found\n", arg)
			anyNotFound = true
		}
		if anyNotFound {
			return 1
		}
	case "eval":
		src := strings.Join(args, " ")
		p := syntax.NewParser()
		file, err := p.Parse(strings.NewReader(src), "")
		if err != nil {
			r.errf("eval: %v\n", err)
			return 1
		}
		r.stmts(ctx, file.StmtList)
		return r.exit
	case "source", ".":
		if len(args) < 1 {
			r.errf("%v: source: need filename\n", pos)
			return 2
		}
		f, err := r.open(ctx, r.relPath(args[0]), os.O_RDONLY, 0, false)
		if err != nil {
			r.errf("source: %v\n", err)
			return 1
		}
		defer f.Close()
		p := syntax.NewParser()
		file, err := p.Parse(f, args[0])
		if err != nil {
			r.errf("source: %v\n", err)
			return 1
		}
		oldParams := r.Params
		r.Params = args[1:]
		oldInSource := r.inSource
		r.inSource = true
		r.stmts(ctx, file.StmtList)

		r.Params = oldParams
		r.inSource = oldInSource
		if code, ok := r.err.(returnStatus); ok {
			r.err = nil
			r.exit = int(code)
		}
		return r.exit
	case "[":
		if len(args) == 0 || args[len(args)-1] != "]" {
			r.errf("%v: [: missing matching ]\n", pos)
			return 2
		}
		args = args[:len(args)-1]
		fallthrough
	case "test":
		parseErr := false
		p := testParser{
			rem: args,
			err: func(err error) {
				r.errf("%v: %v\n", pos, err)
				parseErr = true
			},
		}
		p.next()
		expr := p.classicTest("[", false)
		if parseErr {
			return 2
		}
		return oneIf(r.bashTest(ctx, expr, true) == "")
	case "exec":
		// TODO: Consider syscall.Exec, i.e. actually replacing
		// the process. It's in theory what a shell should do,
		// but in practice it would kill the entire Go process
		// and it's not available on Windows.
		if len(args) == 0 {
			r.keepRedirs = true
			break
		}
		r.exec(ctx, args)
		r.setErr(ShellExitStatus(r.exit))
		return 0
	case "command":
		show := false
		for len(args) > 0 && strings.HasPrefix(args[0], "-") {
			switch args[0] {
			case "-v":
				show = true
			default:
				r.errf("command: invalid option %s\n", args[0])
				return 2
			}
			args = args[1:]
		}
		if len(args) == 0 {
			break
		}
		if !show {
			if isBuiltin(args[0]) {
				return r.builtinCode(ctx, pos, args[0], args[1:])
			}
			r.exec(ctx, args)
			return r.exit
		}
		last := 0
		for _, arg := range args {
			last = 0
			if r.Funcs[arg] != nil || isBuiltin(arg) {
				r.outf("%s\n", arg)
			} else if path, err := exec.LookPath(arg); err == nil {
				r.outf("%s\n", path)
			} else {
				last = 1
			}
		}
		return last
	case "dirs":
		for i := len(r.dirStack) - 1; i >= 0; i-- {
			r.outf("%s", r.dirStack[i])
			if i > 0 {
				r.out(" ")
			}
		}
		r.out("\n")
	case "pushd":
		change := true
		if len(args) > 0 && args[0] == "-n" {
			change = false
			args = args[1:]
		}
		swap := func() string {
			oldtop := r.dirStack[len(r.dirStack)-1]
			top := r.dirStack[len(r.dirStack)-2]
			r.dirStack[len(r.dirStack)-1] = top
			r.dirStack[len(r.dirStack)-2] = oldtop
			return top
		}
		switch len(args) {
		case 0:
			if !change {
				break
			}
			if len(r.dirStack) < 2 {
				r.errf("pushd: no other directory\n")
				return 1
			}
			newtop := swap()
			if code := r.changeDir(newtop); code != 0 {
				return code
			}
			r.builtinCode(ctx, syntax.Pos{}, "dirs", nil)
		case 1:
			if change {
				if code := r.changeDir(args[0]); code != 0 {
					return code
				}
				r.dirStack = append(r.dirStack, r.Dir)
			} else {
				r.dirStack = append(r.dirStack, args[0])
				swap()
			}
			r.builtinCode(ctx, syntax.Pos{}, "dirs", nil)
		default:
			r.errf("pushd: too many arguments\n")
			return 2
		}
	case "popd":
		change := true
		if len(args) > 0 && args[0] == "-n" {
			change = false
			args = args[1:]
		}
		switch len(args) {
		case 0:
			if len(r.dirStack) < 2 {
				r.errf("popd: directory stack empty\n")
				return 1
			}
			oldtop := r.dirStack[len(r.dirStack)-1]
			r.dirStack = r.dirStack[:len(r.dirStack)-1]
			if change {
				newtop := r.dirStack[len(r.dirStack)-1]
				if code := r.changeDir(newtop); code != 0 {
					return code
				}
			} else {
				r.dirStack[len(r.dirStack)-1] = oldtop
			}
			r.builtinCode(ctx, syntax.Pos{}, "dirs", nil)
		default:
			r.errf("popd: invalid argument\n")
			return 2
		}
	case "return":
		if !r.inFunc && !r.inSource {
			r.errf("return: can only be done from a func or sourced script\n")
			return 1
		}
		code := 0
		switch len(args) {
		case 0:
		case 1:
			code = atoi(args[0])
		default:
			r.errf("return: too many arguments\n")
			return 2
		}
		r.setErr(returnStatus(code))
	case "read":
		raw := false
		for len(args) > 0 && strings.HasPrefix(args[0], "-") {
			switch args[0] {
			case "-r":
				raw = true
			default:
				r.errf("read: invalid option %q\n", args[0])
				return 2
			}
			args = args[1:]
		}

		for _, name := range args {
			if !syntax.ValidName(name) {
				r.errf("read: invalid identifier %q\n", name)
				return 2
			}
		}

		line, err := r.readLine(raw)
		if err != nil {
			return 1
		}
		if len(args) == 0 {
			args = append(args, "REPLY")
		}

		values := r.ifsFields(string(line), len(args), raw)
		for i, name := range args {
			val := ""
			if i < len(values) {
				val = values[i]
			}
			r.setVar(ctx, name, nil, Variable{Value: StringVal(val)})
		}

		return 0

	case "getopts":
		if len(args) < 2 {
			r.errf("getopts: usage: getopts optstring name [arg]\n")
			return 2
		}
		optind, _ := strconv.Atoi(r.getVar("OPTIND"))
		if optind-1 != r.optState.argidx {
			if optind < 1 {
				optind = 1
			}
			r.optState = getopts{argidx: optind - 1}
		}
		optstr := args[0]
		name := args[1]
		if !syntax.ValidName(name) {
			r.errf("getopts: invalid identifier: %q\n", name)
			return 2
		}
		args = args[2:]
		if len(args) == 0 {
			args = r.Params
		}
		diagnostics := !strings.HasPrefix(optstr, ":")

		opt, optarg, done := r.optState.Next(optstr, args)

		r.setVarString(ctx, name, string(opt))
		r.delVar("OPTARG")
		switch {
		case opt == '?' && diagnostics && !done:
			r.errf("getopts: illegal option -- %q\n", optarg)
		case opt == ':' && diagnostics:
			r.errf("getopts: option requires an argument -- %q\n", optarg)
		default:
			if optarg != "" {
				r.setVarString(ctx, "OPTARG", optarg)
			}
		}
		if optind-1 != r.optState.argidx {
			r.setVarString(ctx, "OPTIND", strconv.FormatInt(int64(r.optState.argidx+1), 10))
		}

		return oneIf(done)

	case "shopt":
		mode := ""
		posixOpts := false
		for len(args) > 0 && strings.HasPrefix(args[0], "-") {
			switch args[0] {
			case "-s", "-u":
				mode = args[0]
			case "-o":
				posixOpts = true
			case "-p", "-q":
				panic(fmt.Sprintf("unhandled shopt flag: %s", args[0]))
			default:
				r.errf("shopt: invalid option %q\n", args[0])
				return 2
			}
			args = args[1:]
		}
		if len(args) == 0 {
			if !posixOpts {
				for i, name := range bashOptsTable {
					r.printOptLine(name, r.opts[len(shellOptsTable)+i])
				}
				break
			}
			for i, opt := range &shellOptsTable {
				r.printOptLine(opt.name, r.opts[i])
			}
			break
		}
		for _, arg := range args {
			opt := r.optByName(arg, !posixOpts)
			if opt == nil {
				r.errf("shopt: invalid option name %q\n", arg)
				return 1
			}
			switch mode {
			case "-s", "-u":
				*opt = mode == "-s"
			default: // ""
				r.printOptLine(arg, *opt)
			}
		}

	default:
		// "trap", "umask", "alias", "unalias", "fg", "bg",
		panic(fmt.Sprintf("unhandled builtin: %s", name))
	}
	return 0
}

func (r *Runner) printOptLine(name string, enabled bool) {
	status := "off"
	if enabled {
		status = "on"
	}
	r.outf("%s\t%s\n", name, status)
}

func (r *Runner) ifsFields(s string, n int, raw bool) []string {
	type pos struct {
		start, end int
	}
	var fpos []pos

	runes := make([]rune, 0, len(s))
	infield := false
	esc := false
	for _, c := range s {
		if infield {
			if r.ifsRune(c) && (raw || !esc) {
				fpos[len(fpos)-1].end = len(runes)
				infield = false
			}
		} else {
			if !r.ifsRune(c) && (raw || !esc) {
				fpos = append(fpos, pos{start: len(runes), end: -1})
				infield = true
			}
		}
		if c == '\\' {
			if raw || esc {
				runes = append(runes, c)
			}
			esc = !esc
			continue
		}
		runes = append(runes, c)
		esc = false
	}
	if len(fpos) == 0 {
		return nil
	}
	if infield {
		fpos[len(fpos)-1].end = len(runes)
	}

	switch {
	case n == 1:
		// include heading/trailing IFSs
		fpos[0].start, fpos[0].end = 0, len(runes)
		fpos = fpos[:1]
	case n != -1 && n < len(fpos):
		// combine to max n fields
		fpos[n-1].end = fpos[len(fpos)-1].end
		fpos = fpos[:n]
	}

	var fields = make([]string, len(fpos))
	for i, p := range fpos {
		fields[i] = string(runes[p.start:p.end])
	}
	return fields
}

func (r *Runner) readLine(raw bool) ([]byte, error) {
	var line []byte
	esc := false

	for {
		var buf [1]byte
		n, err := r.Stdin.Read(buf[:])
		if n > 0 {
			b := buf[0]
			switch {
			case !raw && b == '\\':
				line = append(line, b)
				esc = !esc
			case !raw && b == '\n' && esc:
				// line continuation
				line = line[len(line)-1:]
				esc = false
			case b == '\n':
				return line, nil
			default:
				line = append(line, b)
				esc = false
			}
		}
		if err == io.EOF && len(line) > 0 {
			return line, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

func (r *Runner) changeDir(path string) int {
	path = r.relPath(path)
	info, err := r.stat(path)
	if err != nil || !info.IsDir() {
		return 1
	}
	if !hasPermissionToDir(info) {
		return 1
	}
	r.Dir = path
	r.Vars["OLDPWD"] = r.Vars["PWD"]
	r.Vars["PWD"] = Variable{Value: StringVal(path)}
	return 0
}

func (r *Runner) relPath(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.Dir, path)
	}
	return filepath.Clean(path)
}

type getopts struct {
	argidx  int
	runeidx int
}

func (g *getopts) Next(optstr string, args []string) (opt rune, optarg string, done bool) {
	if len(args) == 0 || g.argidx >= len(args) {
		return '?', "", true
	}
	arg := []rune(args[g.argidx])
	if len(arg) < 2 || arg[0] != '-' || arg[1] == '-' {
		return '?', "", true
	}

	opts := arg[1:]
	opt = opts[g.runeidx]
	if g.runeidx+1 < len(opts) {
		g.runeidx++
	} else {
		g.argidx++
		g.runeidx = 0
	}

	i := strings.IndexRune(optstr, opt)
	if i < 0 {
		// invalid option
		return '?', string(opt), false
	}

	if i+1 < len(optstr) && optstr[i+1] == ':' {
		if g.argidx >= len(args) {
			// missing argument
			return ':', string(opt), false
		}
		optarg = args[g.argidx]
		g.argidx++
		g.runeidx = 0
	}

	return opt, optarg, false
}
