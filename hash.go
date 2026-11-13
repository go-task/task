package task

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/go-task/task/v3/internal/hash"
	"github.com/go-task/task/v3/taskfile/ast"
)

// reference is a memoized resolution; ok is false for a name that resolves to
// no task.
type reference struct {
	id string
	ok bool
}

func (e *Executor) GetHash(t *ast.Task) (string, error) {
	r := cmp.Or(t.Run, e.Taskfile.Run)
	var h hash.HashFunc
	switch r {
	case "always":
		h = hash.Empty
	case "once":
		h = hash.Name
	case "when_changed":
		h = hash.Hash
	default:
		return "", fmt.Errorf(`task: invalid run "%s"`, r)
	}
	return h(t, e.resolveReference)
}

// resolveReference names what a task reference runs: the definition it
// resolves to, the wildcard match, and what that definition's deps and task
// commands resolve to in turn. The merged Taskfile does not change after
// setup, so each name is looked up once per executor, including a name that
// resolves to no task.
func (e *Executor) resolveReference(name string) (string, bool) {
	id, ok, _ := e.resolveReferenceFrom(name, map[string]bool{})
	return id, ok
}

// resolveReferenceFrom resolves name with stack holding the definitions being
// resolved above it. A definition met again on the stack is named by its site
// alone. No name whose walk reached such a cut is memoized, since its result
// then depends on where the walk entered the cycle.
func (e *Executor) resolveReferenceFrom(name string, stack map[string]bool) (string, bool, bool) {
	if r, ok := e.references.Load(name); ok {
		ref := r.(reference)
		return ref.id, ref.ok, false
	}
	matches, err := e.FindMatchingTasks(&Call{Task: name})
	if err != nil || len(matches) == 0 || matches[0].Task.Location == nil {
		e.references.Store(name, reference{})
		return "", false, false
	}
	t := matches[0].Task
	site := fmt.Sprintf("%s:%d:%d:%s", t.Location.Taskfile, t.Location.Line, t.Location.Column, strings.Join(matches[0].Wildcards, ","))
	if stack[site] {
		return site, true, true
	}
	stack[site] = true
	defer delete(stack, site)

	var refs []string
	cut := false
	add := func(ref string) {
		id, ok, c := e.resolveReferenceFrom(ref, stack)
		if !ok {
			id = ref
		}
		cut = cut || c
		refs = append(refs, id)
	}
	for _, d := range t.Deps {
		if d != nil && d.Task != "" {
			add(d.Task)
		}
	}
	for _, cmd := range t.Cmds {
		if cmd != nil && cmd.Task != "" {
			add(cmd.Task)
		}
	}
	sum := sha256.Sum256([]byte(site + "[" + strings.Join(refs, ",") + "]"))
	id := hex.EncodeToString(sum[:])
	if !cut {
		e.references.Store(name, reference{id: id, ok: true})
	}
	return id, true, cut
}
