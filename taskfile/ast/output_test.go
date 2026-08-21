package ast_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestOutputParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		expect  ast.Output
	}{
		{
			name:    "scalar group output",
			content: "group",
			expect:  ast.Output{Name: "group"},
		},
		{
			name: "group output options",
			content: `group:
  begin: '::group::{{.TASK}}'
  end: '::endgroup::'
  error_only: true
  by_task: true
`,
			expect: ast.Output{
				Name: "group",
				Group: ast.OutputGroup{
					Begin:     "::group::{{.TASK}}",
					End:       "::endgroup::",
					ErrorOnly: true,
					ByTask:    true,
				},
			},
		},
		{
			name:    "group output defaults",
			content: "group: {}\n",
			expect:  ast.Output{Name: "group"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var actual ast.Output
			require.NoError(t, yaml.Unmarshal([]byte(test.content), &actual))
			assert.Equal(t, test.expect, actual)
		})
	}
}
