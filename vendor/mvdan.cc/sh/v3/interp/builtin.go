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
	"path/filepath"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
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

func oneIf(b bool) int {
	if b {
		return 1
	}
	return 0
}

// atoi is just a shorthand for strconv.Atoi that ignores the error,
// just like shells do.
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func (r *Runner) builtinCode(ctx context.Context, pos syntax.Pos, name string, args []string) int {
	switch name {
	case "true", ":":
	case "false":
		return 1
	case "exit":
		r.exitShell = true
		switch len(args) {
		case 0:
			return r.lastExit
		case 1:
			n, err := strconv.Atoi(args[0])
			if err != nil {
				r.errf("invalid exit status code: %q\n", args[0])
				return 2
			}
			return n
		default:
			r.errf("exit cannot take multiple arguments\n")
			return 1
		}
	case "set":
		if err := Params(args...)(r); err != nil {
			r.errf("set: %v\n", err)
			return 2
		}
		r.updateExpandOpts()
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
			if vr := r.lookupVar(arg); vr.IsSet() && vars {
				r.delVar(arg)
				continue
			}
			if _, ok := r.Funcs[arg]; ok && funcs {
				delete(r.Funcs, arg)
			}
		}
	case "echo":
		newline, doExpand := true, false
	echoOpts:
		for len(args) > 0 {
			switch args[0] {
			case "-n":
				newline = false
			case "-e":
				doExpand = true
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
			if doExpand {
				arg, _, _ = expand.Format(r.ecfg, arg, nil)
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
			s, n, err := expand.Format(r.ecfg, format, args)
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
	case "break", "continue":
		if !r.inLoop {
			r.errf("%s is only useful in a loop", name)
			break
		}
		enclosing := &r.breakEnclosing
		if name == "continue" {
			enclosing = &r.contnEnclosing
		}
		switch len(args) {
		case 0:
			*enclosing = 1
		case 1:
			if n, err := strconv.Atoi(args[0]); err == nil {
				*enclosing = n
				break
			}
			fallthrough
		default:
			r.errf("usage: %s [n]\n", name)
			return 2
		}
	case "pwd":
		r.outf("%s\n", r.envGet("PWD"))
	case "cd":
		var path string
		switch len(args) {
		case 0:
			path = r.envGet("HOME")
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
		err := r.bgShells.Wait()
		if _, ok := IsExitStatus(err); err != nil && !ok {
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
			if als, ok := r.alias[arg]; ok && r.opts[optExpandAliases] {
				var buf bytes.Buffer
				if len(als.args) > 0 {
					printer := syntax.NewPrinter()
					printer.Print(&buf, &syntax.CallExpr{
						Args: als.args,
					})
				}
				if als.blank {
					buf.WriteByte(' ')
				}
				r.outf("%s is aliased to `%s'\n", arg, &buf)
				continue
			}
			if _, ok := r.Funcs[arg]; ok {
				r.outf("%s is a function\n", arg)
				continue
			}
			if isBuiltin(arg) {
				r.outf("%s is a shell builtin\n", arg)
				continue
			}
			if path, err := LookPath(expandEnv{r}, arg); err == nil {
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
		r.stmts(ctx, file.Stmts)
		return r.exit
	case "source", ".":
		if len(args) < 1 {
			r.errf("%v: source: need filename\n", pos)
			return 2
		}
		f, err := r.open(ctx, args[0], os.O_RDONLY, 0, false)
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

		// Keep the current versions of some fields we might modify.
		oldParams := r.Params
		oldSourceSetParams := r.sourceSetParams
		oldInSource := r.inSource

		// If we run "source file args...", set said args as parameters.
		// Otherwise, keep the current parameters.
		sourceArgs := len(args[1:]) > 0
		if sourceArgs {
			r.Params = args[1:]
			r.sourceSetParams = false
		}
		// We want to track if the sourced file explicitly sets the
		// paramters.
		r.sourceSetParams = false
		r.inSource = true // know that we're inside a sourced script.
		r.stmts(ctx, file.Stmts)

		// If we modified the parameters and the sourced file didn't
		// explicitly set them, we restore the old ones.
		if sourceArgs && !r.sourceSetParams {
			r.Params = oldParams
		}
		r.sourceSetParams = oldSourceSetParams
		r.inSource = oldInSource

		if code, ok := r.err.(returnStatus); ok {
			r.err = nil
			return int(code)
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
		// TODO: Consider unix.Exec, i.e. actually replacing
		// the process. It's in theory what a shell should do,
		// but in practice it would kill the entire Go process
		// and it's not available on Windows.
		if len(args) == 0 {
			r.keepRedirs = true
			break
		}
		r.exitShell = true
		r.exec(ctx, args)
		return r.exit
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

		values := expand.ReadFields(r.ecfg, string(line), len(args), raw)
		for i, name := range args {
			val := ""
			if i < len(values) {
				val = values[i]
			}
			r.setVar(name, nil, expand.Variable{Kind: expand.String, Str: val})
		}

		return 0

	case "getopts":
		if len(args) < 2 {
			r.errf("getopts: usage: getopts optstring name [arg]\n")
			return 2
		}
		optind, _ := strconv.Atoi(r.envGet("OPTIND"))
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

		r.setVarString(name, string(opt))
		r.delVar("OPTARG")
		switch {
		case opt == '?' && diagnostics && !done:
			r.errf("getopts: illegal option -- %q\n", optarg)
		case opt == ':' && diagnostics:
			r.errf("getopts: option requires an argument -- %q\n", optarg)
		default:
			if optarg != "" {
				r.setVarString("OPTARG", optarg)
			}
		}
		if optind-1 != r.optState.argidx {
			r.setVarString("OPTIND", strconv.FormatInt(int64(r.optState.argidx+1), 10))
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
		r.updateExpandOpts()

	case "alias":
		show := func(name string, als alias) {
			var buf bytes.Buffer
			if len(als.args) > 0 {
				printer := syntax.NewPrinter()
				printer.Print(&buf, &syntax.CallExpr{
					Args: als.args,
				})
			}
			if als.blank {
				buf.WriteByte(' ')
			}
			r.outf("alias %s='%s'\n", name, &buf)
		}

		if len(args) == 0 {
			for name, als := range r.alias {
				show(name, als)
			}
		}
		for _, name := range args {
			i := strings.IndexByte(name, '=')
			if i < 1 { // don't save an empty name
				als, ok := r.alias[name]
				if !ok {
					r.errf("alias: %q not found\n", name)
					continue
				}
				show(name, als)
				continue
			}

			// TODO: parse any CallExpr perhaps, or even any Stmt
			parser := syntax.NewParser()
			var words []*syntax.Word
			src := name[i+1:]
			if err := parser.Words(strings.NewReader(src), func(w *syntax.Word) bool {
				words = append(words, w)
				return true
			}); err != nil {
				r.errf("alias: could not parse %q: %v", src, err)
				continue
			}

			name = name[:i]
			if r.alias == nil {
				r.alias = make(map[string]alias)
			}
			r.alias[name] = alias{
				args:  words,
				blank: strings.TrimRight(src, " \t") != src,
			}
		}
	case "unalias":
		for _, name := range args {
			delete(r.alias, name)
		}

	default:
		// "trap", "umask", "fg", "bg",
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

func (r *Runner) readLine(raw bool) ([]byte, error) {
	var line []byte
	esc := false

	for {
		var buf [1]byte
		n, err := r.stdin.Read(buf[:])
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
	path = r.absPath(path)
	info, err := r.stat(path)
	if err != nil || !info.IsDir() {
		return 1
	}
	if !hasPermissionToDir(info) {
		return 1
	}
	r.Dir = path
	r.Vars["OLDPWD"] = r.Vars["PWD"]
	r.Vars["PWD"] = expand.Variable{Kind: expand.String, Str: path}
	return 0
}

func (r *Runner) absPath(path string) string {
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
