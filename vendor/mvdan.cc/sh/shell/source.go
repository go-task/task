// Copyright (c) 2018, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package shell

import (
	"context"
	"fmt"
	"os"

	"mvdan.cc/sh/expand"
	"mvdan.cc/sh/interp"
	"mvdan.cc/sh/syntax"
)

// SourceFile sources a shell file from disk and returns the variables
// declared in it. It is a convenience function that uses a default shell
// parser, parses a file from disk, and calls SourceNode.
//
// This function should be used with caution, as it can interpret arbitrary
// code. Untrusted shell programs shoudn't be sourced outside of a sandbox
// environment.
func SourceFile(ctx context.Context, path string) (map[string]expand.Variable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open: %v", err)
	}
	defer f.Close()
	file, err := syntax.NewParser().Parse(f, path)
	if err != nil {
		return nil, fmt.Errorf("could not parse: %v", err)
	}
	return SourceNode(ctx, file)
}

// SourceNode sources a shell program from a node and returns the
// variables declared in it. It accepts the same set of node types that
// interp/Runner.Run does.
//
// This function should be used with caution, as it can interpret arbitrary
// code. Untrusted shell programs shoudn't be sourced outside of a sandbox
// environment.
func SourceNode(ctx context.Context, node syntax.Node) (map[string]expand.Variable, error) {
	r, _ := interp.New()
	if err := r.Run(ctx, node); err != nil {
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
