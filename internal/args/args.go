package args

import (
	"errors"
	"strings"

	"github.com/go-task/task/internal/taskfile"
)

var (
	// ErrVariableWithoutTask is returned when variables are given before any task
	ErrVariableWithoutTask = errors.New("task: variable given before any task")
)

// Parse parses command line argument: tasks and vars of each task
func Parse(args ...string) ([]taskfile.Call, error) {
	var calls []taskfile.Call

	for _, arg := range args {
		if !strings.Contains(arg, "=") {
			calls = append(calls, taskfile.Call{Task: arg})
			continue
		}
		if len(calls) < 1 {
			return nil, ErrVariableWithoutTask
		}

		if calls[len(calls)-1].Vars == nil {
			calls[len(calls)-1].Vars = make(taskfile.Vars)
		}

		pair := strings.SplitN(arg, "=", 2)
		calls[len(calls)-1].Vars[pair[0]] = taskfile.Var{Static: pair[1]}
	}
	return calls, nil
}
