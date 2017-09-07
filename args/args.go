package args

import (
	"errors"
	"strings"

	"github.com/go-task/task"
)

var (
	ErrVariableWithoutTask = errors.New("task: variable given before any task")
)

func Parse(args ...string) ([]task.Call, error) {
	var calls []task.Call

	for _, arg := range args {
		if !strings.Contains(arg, "=") {
			calls = append(calls, task.Call{Task: arg})
			continue
		}
		if len(calls) < 1 {
			return nil, ErrVariableWithoutTask
		}

		if calls[len(calls)-1].Vars == nil {
			calls[len(calls)-1].Vars = make(task.Vars)
		}

		pair := strings.SplitN(arg, "=", 2)
		calls[len(calls)-1].Vars[pair[0]] = task.Var{Static: pair[1]}
	}
	return calls, nil
}
