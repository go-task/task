// Copyright (c) 2018, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package shell

import (
	"strings"

	"mvdan.cc/sh/interp"
	"mvdan.cc/sh/syntax"
)

// Expand performs shell expansion on s, using env to resolve variables.
// The expansion will apply to parameter expansions like $var and
// ${#var}, but also to arithmetic expansions like $((var + 3)), and
// command substitutions like $(echo foo).
//
// If env is nil, the current environment variables are used.
//
// Any side effects or modifications to the system are forbidden when
// interpreting the program. This is enforced via whitelists when
// executing programs and opening paths.
func Expand(s string, env func(string) string) (string, error) {
	p := syntax.NewParser()
	src := "<<EOF\n" + s + "\nEOF"
	f, err := p.Parse(strings.NewReader(src), "")
	if err != nil {
		return "", err
	}
	word := f.Stmts[0].Redirs[0].Hdoc
	r := pureRunner()
	if env != nil {
		r.Env = interp.FuncEnviron(env)
	}
	r.Reset()
	fields := r.Fields(word)
	// TODO: runner error
	join := strings.Join(fields, "")
	// since the heredoc implies a trailing newline
	return strings.TrimSuffix(join, "\n"), nil
}
