package args_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/args"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Args            []string
		ExpectedCalls   []*task.Call
		ExpectedGlobals *ast.Vars
	}{
		{
			Args: []string{"task-a", "task-b", "task-c"},
			ExpectedCalls: []*task.Call{
				{Task: "task-a"},
				{Task: "task-b"},
				{Task: "task-c"},
			},
		},
		{
			Args: []string{"task-a", "FOO=bar", "task-b", "task-c", "BAR=baz", "BAZ=foo"},
			ExpectedCalls: []*task.Call{
				{Task: "task-a"},
				{Task: "task-b"},
				{Task: "task-c"},
			},
			ExpectedGlobals: ast.NewVars(
				&ast.VarElement{
					Key: "FOO",
					Value: ast.Var{
						Value: "bar",
					},
				},
				&ast.VarElement{
					Key: "BAR",
					Value: ast.Var{
						Value: "baz",
					},
				},
				&ast.VarElement{
					Key: "BAZ",
					Value: ast.Var{
						Value: "foo",
					},
				},
			),
		},
		{
			Args: []string{"task-a", "CONTENT=with some spaces"},
			ExpectedCalls: []*task.Call{
				{Task: "task-a"},
			},
			ExpectedGlobals: ast.NewVars(
				&ast.VarElement{
					Key: "CONTENT",
					Value: ast.Var{
						Value: "with some spaces",
					},
				},
			),
		},
		{
			Args: []string{"FOO=bar", "task-a", "task-b"},
			ExpectedCalls: []*task.Call{
				{Task: "task-a"},
				{Task: "task-b"},
			},
			ExpectedGlobals: ast.NewVars(
				&ast.VarElement{
					Key: "FOO",
					Value: ast.Var{
						Value: "bar",
					},
				},
			),
		},
		{
			Args:          nil,
			ExpectedCalls: []*task.Call{},
		},
		{
			Args:          []string{},
			ExpectedCalls: []*task.Call{},
		},
		{
			Args:          []string{"FOO=bar", "BAR=baz"},
			ExpectedCalls: []*task.Call{},
			ExpectedGlobals: ast.NewVars(
				&ast.VarElement{
					Key: "FOO",
					Value: ast.Var{
						Value: "bar",
					},
				},
				&ast.VarElement{
					Key: "BAR",
					Value: ast.Var{
						Value: "baz",
					},
				},
			),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("TestArgs%d", i+1), func(t *testing.T) {
			t.Parallel()

			calls, globals := args.Parse(test.Args...)
			assert.Equal(t, test.ExpectedCalls, calls)
			if test.ExpectedGlobals.Len() > 0 || globals.Len() > 0 {
				assert.Equal(t, test.ExpectedGlobals, globals)
				assert.Equal(t, test.ExpectedGlobals, globals)
			}
		})
	}
}
