package task_test

import (
	"testing"

	"github.com/go-task/task"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
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
	)
	tests := []struct {
		content  string
		v        interface{}
		expected interface{}
	}{
		{
			yamlCmd,
			&task.Cmd{},
			&task.Cmd{Cmd: `echo "a string command"`},
		},
		{
			yamlTaskCall,
			&task.Cmd{},
			&task.Cmd{Task: "another-task", Vars: task.Vars{
				"PARAM1": task.Var{Static: "VALUE1"},
				"PARAM2": task.Var{Static: "VALUE2"},
			}},
		},
		{
			yamlDep,
			&task.Dep{},
			&task.Dep{Task: "task-name"},
		},
		{
			yamlTaskCall,
			&task.Dep{},
			&task.Dep{Task: "another-task", Vars: task.Vars{
				"PARAM1": task.Var{Static: "VALUE1"},
				"PARAM2": task.Var{Static: "VALUE2"},
			}},
		},
	}
	for _, test := range tests {
		err := yaml.Unmarshal([]byte(test.content), test.v)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, test.v)
	}
}
