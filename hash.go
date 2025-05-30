package task

import (
	"cmp"
	"fmt"

	"github.com/go-task/task/v3/internal/hash"
	"github.com/go-task/task/v3/taskfile/ast"
)

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
	case "init":
		h = hash.Name  // Run init tasks _once_ only.
	case "exit":
		h = hash.Empty  // Run exit tasks _always_.
	default:
		return "", fmt.Errorf(`task: invalid run "%s"`, r)
	}
	return h(t)
}
