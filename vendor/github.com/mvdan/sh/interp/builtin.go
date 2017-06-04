// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package interp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mvdan/sh/syntax"
)

func isBuiltin(name string) bool {
	switch name {
	case "true", ":", "false", "exit", "set", "shift", "unset",
		"echo", "printf", "break", "continue", "pwd", "cd",
		"wait", "builtin", "trap", "type", "source", "command",
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
		r.args = args
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
		if len(r.args) < n {
			n = len(r.args)
		}
		r.args = r.args[n:]
	case "unset":
		for _, arg := range args {
			r.delVar(arg)
		}
	case "echo":
		newline := true
	opts:
		for len(args) > 0 {
			switch args[0] {
			case "-n":
				newline = false
			case "-e", "-E":
				// TODO: what should be our default?
				// exactly what is the difference in
				// what we write?
			default:
				break opts
			}
			args = args[1:]
		}
		for i, arg := range args {
			if i > 0 {
				r.outf(" ")
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
		var a []interface{}
		for _, arg := range args[1:] {
			a = append(a, arg)
		}
		r.outf(args[0], a...)
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
		if len(args) > 1 {
			r.errf("usage: cd [dir]\n")
			return 2
		}
		var dir string
		if len(args) == 0 {
			dir = r.getVar("HOME")
		} else {
			dir = args[0]
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(r.Dir, dir)
		}
		_, err := os.Stat(dir)
		if err != nil {
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
		r2.File = file
		r2.Run()
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
	case "trap", "source", "command", "pushd", "popd",
		"umask", "alias", "unalias", "fg", "bg", "getopts":
		r.runErr(pos, "unhandled builtin: %s", name)
	}
	return 0
}
