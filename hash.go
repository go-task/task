package task

import (
	"fmt"

	"github.com/go-task/task/v3/internal/hash"
	"github.com/go-task/task/v3/taskfile"
)

func (e *Executor) GetHash(t *taskfile.Task) (string, error) {
	r := t.Run
	if r == "" {
		r = e.Taskfile.Run
	}

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
	return h(t)
}
