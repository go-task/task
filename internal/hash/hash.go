package hash

import (
	"fmt"
	"strings"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/go-task/task/v3/taskfile/ast"
)

// Resolve names what a task reference runs: the definition it resolves to
// and, in turn, everything that definition references. It reports false for
// a reference it cannot resolve statically, such as a templated name.
type Resolve func(name string) (string, bool)

type HashFunc func(*ast.Task, Resolve) (string, error)

func Empty(*ast.Task, Resolve) (string, error) {
	return "", nil
}

// Name keys a task by what it runs: where it is defined, the wildcard match
// it was called as, and what its deps and task commands resolve to. Calls
// reached through different include paths share a key when they run the same
// tasks, and stay apart when a reference resolves to a different task on each
// path.
func Name(t *ast.Task, resolve Resolve) (string, error) {
	c := resolved(t, resolve)
	var refs []string
	for _, d := range c.Deps {
		if d != nil && d.Task != "" {
			refs = append(refs, d.Task)
		}
	}
	for _, cmd := range c.Cmds {
		if cmd != nil && cmd.Task != "" {
			refs = append(refs, cmd.Task)
		}
	}
	return fmt.Sprintf("%s[%s]", identity(t), strings.Join(refs, ",")), nil
}

// Hash keys a task call as Name does and by the task as compiled for the
// call, so calls with different variables stay apart. Aliases, which do not
// change what a call runs, are left out.
func Hash(t *ast.Task, resolve Resolve) (string, error) {
	c := resolved(t, resolve)
	c.Aliases = nil
	h, err := hashstructure.Hash(c, hashstructure.FormatV2, nil)
	return fmt.Sprintf("%s:%d", identity(t), h), err
}

// resolved is a copy of t whose task references name what they resolve to. A
// reference that cannot be resolved keeps its name, which carries the include
// path it was reached through, so it can keep two calls apart but never merge
// them.
func resolved(t *ast.Task, resolve Resolve) *ast.Task {
	name := func(n string) string {
		if r, ok := resolve(n); ok {
			return r
		}
		return n
	}
	c := *t
	c.Deps = make([]*ast.Dep, len(t.Deps))
	for i, d := range t.Deps {
		if d != nil && d.Task != "" {
			dc := *d
			dc.Task = name(d.Task)
			d = &dc
		}
		c.Deps[i] = d
	}
	c.Cmds = make([]*ast.Cmd, len(t.Cmds))
	for i, cmd := range t.Cmds {
		if cmd != nil && cmd.Task != "" {
			cc := *cmd
			cc.Task = name(cmd.Task)
			cmd = &cc
		}
		c.Cmds[i] = cmd
	}
	return &c
}

// identity names a task by where it is defined and which wildcard match it
// was called as. LocalName cannot stand in for the definition: Namespace holds
// only the last include level merged, so a name reached through a deeper path
// keeps its intermediate segments and differs from path to path.
func identity(t *ast.Task) string {
	var match string
	if t.Vars != nil {
		if m, ok := t.Vars.Get("MATCH"); ok {
			if ms, ok := m.Value.([]string); ok {
				match = strings.Join(ms, ",")
			}
		}
	}
	if t.Location == nil {
		return fmt.Sprintf("%s:%s", t.LocalName(), match)
	}
	return fmt.Sprintf("%s:%d:%d:%s", t.Location.Taskfile, t.Location.Line, t.Location.Column, match)
}
