package args_test

import (
	"fmt"
	"testing"

	"github.com/go-task/task/v2/internal/args"
	"github.com/go-task/task/v2/internal/taskfile"

	"github.com/stretchr/testify/assert"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		Args     []string
		Expected []taskfile.Call
		Err      error
	}{
		{
			Args: []string{"task-a", "task-b", "task-c"},
			Expected: []taskfile.Call{
				{Task: "task-a"},
				{Task: "task-b"},
				{Task: "task-c"},
			},
		},
		{
			Args: []string{"task-a", "FOO=bar", "task-b", "task-c", "BAR=baz", "BAZ=foo"},
			Expected: []taskfile.Call{
				{
					Task: "task-a",
					Vars: taskfile.Vars{
						"FOO": taskfile.Var{Static: "bar"},
					},
				},
				{Task: "task-b"},
				{
					Task: "task-c",
					Vars: taskfile.Vars{
						"BAR": taskfile.Var{Static: "baz"},
						"BAZ": taskfile.Var{Static: "foo"},
					},
				},
			},
		},
		{
			Args: []string{"task-a", "CONTENT=with some spaces"},
			Expected: []taskfile.Call{
				{
					Task: "task-a",
					Vars: taskfile.Vars{
						"CONTENT": taskfile.Var{Static: "with some spaces"},
					},
				},
			},
		},
		{
			Args: []string{"FOO=bar", "task-a"},
			Err:  args.ErrVariableWithoutTask,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("TestArgs%d", i+1), func(t *testing.T) {
			calls, err := args.Parse(test.Args...)
			assert.Equal(t, test.Err, err)
			assert.Equal(t, test.Expected, calls)
		})
	}
}
