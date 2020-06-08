package args

import (
	"strings"

	"github.com/go-task/task/v2/internal/taskfile"
)

// Parse parses command line argument: tasks and vars of each task
func Parse(args ...string) ([]taskfile.Call, *taskfile.Vars) {
	var calls []taskfile.Call
	var globals *taskfile.Vars

	for _, arg := range args {
		if !strings.Contains(arg, "=") {
			calls = append(calls, taskfile.Call{Task: arg})
			continue
		}

		if globals == nil {
			globals = &taskfile.Vars{}
		}
		name, value := splitVar(arg)
		globals.Set(name, taskfile.Var{Static: value})
	}

	if len(calls) == 0 {
		calls = append(calls, taskfile.Call{Task: "default"})
	}

	return calls, globals
}

func splitVar(s string) (string, string) {
	pair := strings.SplitN(s, "=", 2)
	return pair[0], pair[1]
}
