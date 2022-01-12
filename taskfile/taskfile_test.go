package taskfile_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/taskfile"
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
		v        interface{}
		expected interface{}
	}{
		{
			yamlCmd,
			&taskfile.Cmd{},
			&taskfile.Cmd{Cmd: `echo "a string command"`},
		},
		{
			yamlTaskCall,
			&taskfile.Cmd{},
			&taskfile.Cmd{Task: "another-task", Vars: &taskfile.Vars{
				Keys: []string{"PARAM1", "PARAM2"},
				Mapping: map[string]taskfile.Var{
					"PARAM1": taskfile.Var{Static: "VALUE1"},
					"PARAM2": taskfile.Var{Static: "VALUE2"},
				},
			}},
		},
		{
			yamlDeferredCmd,
			&taskfile.Cmd{},
			&taskfile.Cmd{Cmd: "echo 'test'", Defer: true},
		},
		{
			yamlDeferredCall,
			&taskfile.Cmd{},
			&taskfile.Cmd{Task: "some_task", Vars: &taskfile.Vars{
				Keys: []string{"PARAM1"},
				Mapping: map[string]taskfile.Var{
					"PARAM1": taskfile.Var{Static: "var"},
				},
			}, Defer: true},
		},
		{
			yamlDep,
			&taskfile.Dep{},
			&taskfile.Dep{Task: "task-name"},
		},
		{
			yamlTaskCall,
			&taskfile.Dep{},
			&taskfile.Dep{Task: "another-task", Vars: &taskfile.Vars{
				Keys: []string{"PARAM1", "PARAM2"},
				Mapping: map[string]taskfile.Var{
					"PARAM1": taskfile.Var{Static: "VALUE1"},
					"PARAM2": taskfile.Var{Static: "VALUE2"},
				},
			}},
		},
	}
	for _, test := range tests {
		err := yaml.Unmarshal([]byte(test.content), test.v)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, test.v)
	}
}
