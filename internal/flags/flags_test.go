package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestValidateOutputOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		output    ast.Output
		wantError string
	}{
		{
			name:   "group options with group output",
			output: ast.Output{Name: "group", Group: ast.OutputGroup{Begin: "begin", End: "end", ErrorOnly: true}},
		},
		{
			name:      "group option without group output",
			output:    ast.Output{Name: "interleaved", Group: ast.OutputGroup{Begin: "begin"}},
			wantError: "--output-group-begin without --output=group",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := validateOutputOptions(test.output)
			if test.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantError)
		})
	}
}

func TestValidateTUIOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		enabled   bool
		status    string
		navigator string
		wantError string
	}{
		{name: "TUI without options", enabled: true},
		{name: "TUI with options", enabled: true, status: "labels", navigator: "tree"},
		{name: "status without TUI", status: "labels", wantError: "--tui-status without --tui"},
		{name: "navigator without TUI", navigator: "tree", wantError: "--tui-task-navigator without --tui"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := validateTUIOptions(test.enabled, test.status, test.navigator)
			if test.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantError)
		})
	}
}
