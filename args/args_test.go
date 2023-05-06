package args_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/args"
	"github.com/go-task/task/v3/internal/orderedmap"
	"github.com/go-task/task/v3/taskfile"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		Args            []string
		ExpectedCalls   []taskfile.Call
		ExpectedGlobals *taskfile.Vars
	}{
		{
			Args: []string{"task-a", "task-b", "task-c"},
			ExpectedCalls: []taskfile.Call{
				{Task: "task-a", Direct: true},
				{Task: "task-b", Direct: true},
				{Task: "task-c", Direct: true},
			},
		},
		{
			Args: []string{"task-a", "FOO=bar", "task-b", "task-c", "BAR=baz", "BAZ=foo"},
			ExpectedCalls: []taskfile.Call{
				{Task: "task-a", Direct: true},
				{Task: "task-b", Direct: true},
				{Task: "task-c", Direct: true},
			},
			ExpectedGlobals: &taskfile.Vars{
				OrderedMap: orderedmap.FromMapWithOrder(
					map[string]taskfile.Var{
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
			ExpectedCalls: []taskfile.Call{
				{Task: "task-a", Direct: true},
			},
			ExpectedGlobals: &taskfile.Vars{
				OrderedMap: orderedmap.FromMapWithOrder(
					map[string]taskfile.Var{
						"CONTENT": {Value: "with some spaces"},
					},
					[]string{"CONTENT"},
				),
			},
		},
		{
			Args: []string{"FOO=bar", "task-a", "task-b"},
			ExpectedCalls: []taskfile.Call{
				{Task: "task-a", Direct: true},
				{Task: "task-b", Direct: true},
			},
			ExpectedGlobals: &taskfile.Vars{
				OrderedMap: orderedmap.FromMapWithOrder(
					map[string]taskfile.Var{
						"FOO": {Value: "bar"},
					},
					[]string{"FOO"},
				),
			},
		},
		{
			Args:          nil,
			ExpectedCalls: []taskfile.Call{},
		},
		{
			Args:          []string{},
			ExpectedCalls: []taskfile.Call{},
		},
		{
			Args:          []string{"FOO=bar", "BAR=baz"},
			ExpectedCalls: []taskfile.Call{},
			ExpectedGlobals: &taskfile.Vars{
				OrderedMap: orderedmap.FromMapWithOrder(
					map[string]taskfile.Var{
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
