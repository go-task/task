package args_test

import (
	"fmt"
	"testing"

	"github.com/go-task/task"
	"github.com/go-task/task/internal/args"

	"github.com/stretchr/testify/assert"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		Args     []string
		Expected []task.Call
		Err      error
	}{
		{
			Args: []string{"task-a", "task-b", "task-c"},
			Expected: []task.Call{
				{Task: "task-a"},
				{Task: "task-b"},
				{Task: "task-c"},
			},
		},
		{
			Args: []string{"task-a", "FOO=bar", "task-b", "task-c", "BAR=baz", "BAZ=foo"},
			Expected: []task.Call{
				{
					Task: "task-a",
					Vars: task.Vars{
						"FOO": task.Var{Static: "bar"},
					},
				},
				{Task: "task-b"},
				{
					Task: "task-c",
					Vars: task.Vars{
						"BAR": task.Var{Static: "baz"},
						"BAZ": task.Var{Static: "foo"},
					},
				},
			},
		},
		{
			Args: []string{"task-a", "CONTENT=with some spaces"},
			Expected: []task.Call{
				{
					Task: "task-a",
					Vars: task.Vars{
						"CONTENT": task.Var{Static: "with some spaces"},
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
