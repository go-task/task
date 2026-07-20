package ast

import (
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/go-task/task/v3/errors"
)

// The tables below mirror the unexported posixOptsTable and bashOptsTable in
// mvdan.cc/sh/v3/interp (see interp/api.go:
// https://github.com/mvdan/sh/blob/master/interp/api.go). They must be kept
// in sync whenever the mvdan.cc/sh dependency is updated.

// validSetOptions contains the one-character flag and full name forms of the
// POSIX shell options accepted by "set" (posixOptsTable).
var validSetOptions = []string{
	"a", "allexport",
	"e", "errexit",
	"n", "noexec",
	"f", "noglob",
	"u", "nounset",
	"x", "xtrace",
	"pipefail",
}

// validShoptOptions contains the Bash shell options accepted by "shopt" (the
// entries of bashOptsTable with supported: true).
var validShoptOptions = []string{
	"dotglob",
	"expand_aliases",
	"extglob",
	"globstar",
	"nocaseglob",
	"nullglob",
}

// checkSetOptions checks that every entry of a "set" list is a POSIX shell
// option supported by the interpreter so that typos and unsupported options
// are reported when the Taskfile is parsed instead of when a command runs.
func checkSetOptions(node *yaml.Node, opts []string) error {
	for _, opt := range opts {
		if slices.Contains(validSetOptions, opt) {
			continue
		}
		if slices.Contains(validShoptOptions, opt) {
			return errors.NewTaskfileDecodeError(nil, node).WithMessage(`invalid set option %q (did you mean to put it in "shopt"?)`, opt)
		}
		return errors.NewTaskfileDecodeError(nil, node).WithMessage("invalid set option %q", opt)
	}
	return nil
}

// checkShoptOptions checks that every entry of a "shopt" list is a Bash shell
// option supported by the interpreter so that typos and unsupported options
// are reported when the Taskfile is parsed instead of when a command runs.
func checkShoptOptions(node *yaml.Node, opts []string) error {
	for i, opt := range opts {
		// A "-o" entry makes shopt address the "set" builtin options instead
		// (e.g. `shopt: ["-o", "pipefail"]` runs `shopt -s -o pipefail`), so
		// the remaining entries are validated as set options.
		if opt == "-o" {
			return checkSetOptions(node, opts[i+1:])
		}
		if slices.Contains(validShoptOptions, opt) {
			continue
		}
		if slices.Contains(validSetOptions, opt) {
			return errors.NewTaskfileDecodeError(nil, node).WithMessage(`invalid shopt option %q (did you mean to put it in "set"?)`, opt)
		}
		return errors.NewTaskfileDecodeError(nil, node).WithMessage("invalid shopt option %q", opt)
	}
	return nil
}
