package args

import (
	"strings"

	"github.com/go-task/task/v3/taskfile"
)

// Parse parses command line argument: tasks and global variables
func Parse(args ...string) ([]taskfile.Call, *taskfile.Vars) {
	calls := []taskfile.Call{}
	globals := &taskfile.Vars{}

	for _, arg := range args {
		if !strings.Contains(arg, "=") {
			calls = append(calls, taskfile.Call{Task: arg, Direct: true})
			continue
		}

		name, value := splitVar(arg)
		globals.Set(name, taskfile.Var{Value: value})
	}

	return calls, globals
}

func splitVar(s string) (string, string) {
	pair := strings.SplitN(s, "=", 2)
	return pair[0], pair[1]
}
