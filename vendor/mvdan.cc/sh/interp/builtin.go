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
		"pushd", "popd", "umask", "alias", "unalias", "fg", "bg",
		"getopts", "eval", "test", "[":
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
		var dir string
		switch len(args) {
		case 0:
			dir = r.getVar("HOME")
		case 1:
			dir = args[0]
		default:
			r.errf("usage: cd [dir]\n")
			return 2
		}
		dir = r.relPath(dir)
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return 1
		}
		r.Dir = dir
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
		f, err := os.Open(r.relPath(args[0]))
		if err != nil {
			r.errf("eval: %v\n", err)
			return 1
		}
		defer f.Close()
		p := syntax.NewParser()
		file, err := p.Parse(f, args[0])
		if err != nil {
			r.errf("eval: %v\n", err)
			return 1
		}
		r2 := *r
		r2.Params = args[1:]
		r2.Reset()
		r2.Run(file)
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
	case "trap", "command", "pushd", "popd",
		"umask", "alias", "unalias", "fg", "bg", "getopts":
		r.runErr(pos, "unhandled builtin: %s", name)
	}
	return 0
}

func (r *Runner) relPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(r.Dir, path)
}
