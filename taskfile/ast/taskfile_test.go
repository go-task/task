package ast_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestCmdParse(t *testing.T) {
	t.Parallel()

	const (
		yamlCmd      = `echo "a string command"`
		yamlDep      = `"task-name"`
		yamlTaskCall = `
task: another-task
vars:
  PARAM1: VALUE1
  PARAM2: VALUE2
`
		yamlDeferredCall            = `defer: { task: some_task, vars: { PARAM1: "var" } }`
		yamlDeferredCallWithTimeout = `defer: { task: some_task, timeout: 1s }`
		yamlDeferredCmd             = `defer: echo 'test'`
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
				Task: "another-task",
				Vars: ast.NewVars(
					&ast.VarElement{
						Key: "PARAM1",
						Value: ast.Var{
							Value: "VALUE1",
						},
					},
					&ast.VarElement{
						Key: "PARAM2",
						Value: ast.Var{
							Value: "VALUE2",
						},
					},
				),
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
				Task: "some_task",
				Vars: ast.NewVars(
					&ast.VarElement{
						Key: "PARAM1",
						Value: ast.Var{
							Value: "var",
						},
					},
				),
				Defer: true,
			},
		},
		{
			yamlDeferredCallWithTimeout,
			&ast.Cmd{},
			&ast.Cmd{Task: "some_task", Defer: true, Timeout: time.Second},
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
				Task: "another-task",
				Vars: ast.NewVars(
					&ast.VarElement{
						Key: "PARAM1",
						Value: ast.Var{
							Value: "VALUE1",
						},
					},
					&ast.VarElement{
						Key: "PARAM2",
						Value: ast.Var{
							Value: "VALUE2",
						},
					},
				),
			},
		},
	}
	for _, test := range tests {
		err := yaml.Unmarshal([]byte(test.content), test.v)
		require.NoError(t, err)
		assert.Equal(t, test.expected, test.v)
	}
}

func TestDeferredTaskTimeoutParseError(t *testing.T) {
	t.Parallel()

	var cmd ast.Cmd
	err := yaml.Unmarshal([]byte(`defer: { task: some_task, timeout: invalid }`), &cmd)
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid timeout format")
}

func TestTimeoutParseError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		message string
	}{
		{
			name:    "unparsable duration",
			content: `{cmd: echo, timeout: invalid}`,
			message: "invalid timeout format",
		},
		{
			name:    "zero duration",
			content: `{cmd: echo, timeout: 0s}`,
			message: "timeout must be greater than zero",
		},
		{
			name:    "negative duration",
			content: `{cmd: echo, timeout: -1s}`,
			message: "timeout must be greater than zero",
		},
		{
			name:    "negative duration on a deferred task",
			content: `{defer: {task: some_task, timeout: -5m}}`,
			message: "timeout must be greater than zero",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var cmd ast.Cmd
			err := yaml.Unmarshal([]byte(test.content), &cmd)
			require.Error(t, err)
			assert.ErrorContains(t, err, test.message)
		})
	}
}
