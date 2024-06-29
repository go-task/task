package ast_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/omap"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestCmdParse(t *testing.T) {
	const (
		yamlCmd      = `echo "a string command"`
		yamlDep      = `"task-name"`
		yamlTaskCall = `
task: another-task
vars:
  PARAM1: VALUE1
  PARAM2: VALUE2
`
		yamlDeferredCall = `defer: { task: some_task, vars: { PARAM1: "var" } }`
		yamlDeferredCmd  = `defer: echo 'test'`
	)
	tests := []struct {
		content  string
		v        any
		expected any
	}{
		{
			yamlCmd,
			&ast.Cmd{},
			&ast.Cmd{Cmd: `echo "a string command"`},
		},
		{
			yamlTaskCall,
			&ast.Cmd{},
			&ast.Cmd{
				Task: "another-task", Vars: &ast.Vars{
					OrderedMap: omap.FromMapWithOrder(
						map[string]ast.Var{
							"PARAM1": {Value: "VALUE1"},
							"PARAM2": {Value: "VALUE2"},
						},
						[]string{"PARAM1", "PARAM2"},
					),
				},
			},
		},
		{
			yamlDeferredCmd,
			&ast.Cmd{},
			&ast.Cmd{Cmd: "echo 'test'", Defer: true},
		},
		{
			yamlDeferredCall,
			&ast.Cmd{},
			&ast.Cmd{
				Task: "some_task", Vars: &ast.Vars{
					OrderedMap: omap.FromMapWithOrder(
						map[string]ast.Var{
							"PARAM1": {Value: "var"},
						},
						[]string{"PARAM1"},
					),
				},
				Defer: true,
			},
		},
		{
			yamlDep,
			&ast.Dep{},
			&ast.Dep{Task: "task-name"},
		},
		{
			yamlTaskCall,
			&ast.Dep{},
			&ast.Dep{
				Task: "another-task", Vars: &ast.Vars{
					OrderedMap: omap.FromMapWithOrder(
						map[string]ast.Var{
							"PARAM1": {Value: "VALUE1"},
							"PARAM2": {Value: "VALUE2"},
						},
						[]string{"PARAM1", "PARAM2"},
					),
				},
			},
		},
	}
	for _, test := range tests {
		err := yaml.Unmarshal([]byte(test.content), test.v)
		require.NoError(t, err)
		assert.Equal(t, test.expected, test.v)
	}
}
