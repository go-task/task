// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
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
		"return":
		return true
	}
	return false
}

func (r *Runner) builtinCode(pos syntax.Pos, name string, args []string) int {
	switch name {
	case "true", ":":
	case "false":
		return 1
	case "exit":
		switch len(args) {
		case 0:
		case 1:
			if n, err := strconv.Atoi(args[0]); err != nil {
				r.runErr(pos, "invalid exit code: %q", args[0])
			} else {
				r.exit = n
			}
		default:
			r.runErr(pos, "exit cannot take multiple arguments")
		}
		r.lastExit()
		return r.exit
	case "set":
		rest, err := r.FromArgs(args...)
		if err != nil {
			r.errf("set: %v", err)
			return 2
		}
		r.Params = rest
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
		for _, arg := range args {
			r.delVar(arg)
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
				r.outf(" ")
			}
			if expand {
				arg = r.expand(arg, true)
			}
			r.outf("%s", arg)
		}
		if newline {
			r.outf("\n")
		}
	case "printf":
		if len(args) == 0 {
			r.errf("usage: printf format [arguments]\n")
			return 2
		}
		r.outf("%s", r.expand(args[0], false, args[1:]...))
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
			r.runErr(pos, "wait with args not handled yet")
			break
		}
		r.bgShells.Wait()
	case "builtin":
		if len(args) < 1 {
			break
		}
		if !isBuiltin(args[0]) {
			return 1
		}
		return r.builtinCode(pos, args[0], args[1:])
	case "type":
		anyNotFound := false
		for _, arg := range args {
			if _, ok := r.funcs[arg]; ok {
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
		r2 := *r
		r2.Reset()
		r2.Run(file)
		return r2.exit
	case "source", ".":
		if len(args) < 1 {
			r.runErr(pos, "source: need filename")
		}
		f, err := r.open(r.relPath(args[0]), os.O_RDONLY, 0, false)
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
		r2 := *r
		r2.Params = args[1:]
		r2.Reset()
		r2.canReturn = true
		r2.Run(file)
		if code, ok := r2.err.(returnCode); ok {
			r2.exit = int(code)
		}
		return r2.exit
	case "[":
		if len(args) == 0 || args[len(args)-1] != "]" {
			r.runErr(pos, "[: missing matching ]")
			break
		}
		args = args[:len(args)-1]
		fallthrough
	case "test":
		p := testParser{
			rem: args,
			err: func(format string, a ...interface{}) {
				r.runErr(pos, format, a...)
			},
		}
		p.next()
		expr := p.classicTest("[", false)
		return oneIf(r.bashTest(expr) == "")
	case "exec":
		// TODO: Consider syscall.Exec, i.e. actually replacing
		// the process. It's in theory what a shell should do,
		// but in practice it would kill the entire Go process
		// and it's not available on Windows.
		if len(args) == 0 {
			// TODO: different behavior, apparently
			break
		}
		r.exec(args[0], args[1:])
		r.lastExit()
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
				return r.builtinCode(pos, args[0], args[1:])
			}
			r.exec(args[0], args[1:])
			return r.exit
		}
		last := 0
		for _, arg := range args {
			last = 0
			if r.funcs[arg] != nil || isBuiltin(arg) {
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
				r.outf(" ")
			}
		}
		r.outf("\n")
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
			r.builtinCode(syntax.Pos{}, "dirs", nil)
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
			r.builtinCode(syntax.Pos{}, "dirs", nil)
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
			r.builtinCode(syntax.Pos{}, "dirs", nil)
		default:
			r.errf("popd: invdalid argument\n")
			return 2
		}
	case "return":
		if !r.canReturn {
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
		r.setErr(returnCode(code))
	default:
		// "trap", "umask", "alias", "unalias", "fg", "bg",
		// "getopts"
		r.runErr(pos, "unhandled builtin: %s", name)
	}
	return 0
}

func (r *Runner) changeDir(path string) int {
	path = r.relPath(path)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return 1
	}
	if !hasPermissionToDir(info) {
		return 1
	}
	r.Dir = path
	r.vars["PWD"] = path
	return 0
}

func (r *Runner) relPath(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.Dir, path)
	}
	return filepath.Clean(path)
}
