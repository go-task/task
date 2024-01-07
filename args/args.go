package args

import (
	"strings"

	"github.com/go-task/task/v3/taskfile/ast"
)

// Parse parses command line argument: tasks and global variables
func Parse(args ...string) ([]ast.Call, *ast.Vars) {
	calls := []ast.Call{}
	globals := &ast.Vars{}

	for _, arg := range args {
		if !strings.Contains(arg, "=") {
			calls = append(calls, ast.Call{Task: arg})
			continue
		}

		name, value := splitVar(arg)
		globals.Set(name, ast.Var{Value: value})
	}

	return calls, globals
}

func splitVar(s string) (string, string) {
	pair := strings.SplitN(s, "=", 2)
	return pair[0], pair[1]
}
