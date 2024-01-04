package args_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/args"
	"github.com/go-task/task/v3/internal/omap"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		Args            []string
		ExpectedCalls   []ast.Call
		ExpectedGlobals *ast.Vars
	}{
		{
			Args: []string{"task-a", "task-b", "task-c"},
			ExpectedCalls: []ast.Call{
				{Task: "task-a"},
				{Task: "task-b"},
				{Task: "task-c"},
			},
		},
		{
			Args: []string{"task-a", "FOO=bar", "task-b", "task-c", "BAR=baz", "BAZ=foo"},
			ExpectedCalls: []ast.Call{
				{Task: "task-a"},
				{Task: "task-b"},
				{Task: "task-c"},
			},
			ExpectedGlobals: &ast.Vars{
				OrderedMap: omap.FromMapWithOrder(
					map[string]ast.Var{
						"FOO": {Value: "bar"},
						"BAR": {Value: "baz"},
						"BAZ": {Value: "foo"},
					},
					[]string{"FOO", "BAR", "BAZ"},
				),
			},
		},
		{
			Args: []string{"task-a", "CONTENT=with some spaces"},
			ExpectedCalls: []ast.Call{
				{Task: "task-a"},
			},
			ExpectedGlobals: &ast.Vars{
				OrderedMap: omap.FromMapWithOrder(
					map[string]ast.Var{
						"CONTENT": {Value: "with some spaces"},
					},
					[]string{"CONTENT"},
				),
			},
		},
		{
			Args: []string{"FOO=bar", "task-a", "task-b"},
			ExpectedCalls: []ast.Call{
				{Task: "task-a"},
				{Task: "task-b"},
			},
			ExpectedGlobals: &ast.Vars{
				OrderedMap: omap.FromMapWithOrder(
					map[string]ast.Var{
						"FOO": {Value: "bar"},
					},
					[]string{"FOO"},
				),
			},
		},
		{
			Args:          nil,
			ExpectedCalls: []ast.Call{},
		},
		{
			Args:          []string{},
			ExpectedCalls: []ast.Call{},
		},
		{
			Args:          []string{"FOO=bar", "BAR=baz"},
			ExpectedCalls: []ast.Call{},
			ExpectedGlobals: &ast.Vars{
				OrderedMap: omap.FromMapWithOrder(
					map[string]ast.Var{
						"FOO": {Value: "bar"},
						"BAR": {Value: "baz"},
					},
					[]string{"FOO", "BAR"},
				),
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("TestArgs%d", i+1), func(t *testing.T) {
			calls, globals := args.Parse(test.Args...)
			assert.Equal(t, test.ExpectedCalls, calls)
			if test.ExpectedGlobals.Len() > 0 || globals.Len() > 0 {
				assert.Equal(t, test.ExpectedGlobals.Keys(), globals.Keys())
				assert.Equal(t, test.ExpectedGlobals.Values(), globals.Values())
			}
		})
	}
}
