// Copyright (c) 2018, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"mvdan.cc/sh/interp"
	"mvdan.cc/sh/syntax"
)

// SourceFile sources a shell file from disk and returns the variables
// declared in it.
//
// A default parser is used; to set custom options, use SourceNode
// instead.
func SourceFile(path string) (map[string]interp.Variable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open: %v", err)
	}
	defer f.Close()
	p := syntax.NewParser()
	file, err := p.Parse(f, path)
	if err != nil {
		return nil, fmt.Errorf("could not parse: %v", err)
	}
	return SourceNode(file)
}

// purePrograms holds a list of common programs that do not have side
// effects, or otherwise cannot modify or harm the system that runs
// them.
var purePrograms = []string{
	// string handling
	"sed", "grep", "tr", "cut", "cat", "head", "tail", "seq", "yes",
	"wc",
	// paths
	"ls", "pwd", "basename", "realpath",
	// others
	"env", "sleep", "uniq", "sort",
}

var pureRunnerTimeout = 2 * time.Second

func pureRunner() *interp.Runner {
	r := &interp.Runner{}
	// forbid executing programs that might cause trouble
	r.Exec = func(ctx interp.Ctxt, path string, args []string) error {
		for _, name := range purePrograms {
			if args[0] == name {
				return interp.DefaultExec(ctx, path, args)
			}
		}
		return fmt.Errorf("program not in whitelist: %s", args[0])
	}
	// forbid opening any real files
	r.Open = interp.OpenDevImpls(func(ctx interp.Ctxt, path string, flags int, mode os.FileMode) (io.ReadWriteCloser, error) {
		return nil, fmt.Errorf("cannot open path: %s", ctx.UnixPath(path))
	})
	return r
}

// SourceNode sources a shell program from a node and returns the
// variables declared in it.
//
// Any side effects or modifications to the system are forbidden when
// interpreting the program. This is enforced via whitelists when
// executing programs and opening paths. The interpreter also has a timeout of
// two seconds.
func SourceNode(node syntax.Node) (map[string]interp.Variable, error) {
	r := pureRunner()
	r.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), pureRunnerTimeout)
	defer cancel()
	r.Context = ctx
	if err := r.Run(node); err != nil {
		return nil, fmt.Errorf("could not run: %v", err)
	}
	// delete the internal shell vars that the user is not
	// interested in
	delete(r.Vars, "PWD")
	delete(r.Vars, "HOME")
	delete(r.Vars, "PATH")
	delete(r.Vars, "IFS")
	delete(r.Vars, "OPTIND")
	return r.Vars, nil
}
