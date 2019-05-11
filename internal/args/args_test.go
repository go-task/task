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
		Args            []string
		ExpectedCalls   []taskfile.Call
		ExpectedGlobals taskfile.Vars
	}{
		{
			Args: []string{"task-a", "task-b", "task-c"},
			ExpectedCalls: []taskfile.Call{
				{Task: "task-a"},
				{Task: "task-b"},
				{Task: "task-c"},
			},
		},
		{
			Args: []string{"task-a", "FOO=bar", "task-b", "task-c", "BAR=baz", "BAZ=foo"},
			ExpectedCalls: []taskfile.Call{
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
			ExpectedCalls: []taskfile.Call{
				{
					Task: "task-a",
					Vars: taskfile.Vars{
						"CONTENT": taskfile.Var{Static: "with some spaces"},
					},
				},
			},
		},
		{
			Args: []string{"FOO=bar", "task-a", "task-b"},
			ExpectedCalls: []taskfile.Call{
				{Task: "task-a"},
				{Task: "task-b"},
			},
			ExpectedGlobals: taskfile.Vars{
				"FOO": {Static: "bar"},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("TestArgs%d", i+1), func(t *testing.T) {
			calls, globals := args.Parse(test.Args...)
			assert.Equal(t, test.ExpectedCalls, calls)
			assert.Equal(t, test.ExpectedGlobals, globals)
		})
	}
}
