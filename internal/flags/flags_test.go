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
			name:   "TUI options with TUI output",
			output: ast.Output{Name: "tui", TUI: ast.OutputTUI{Status: "labels", TaskNavigator: "tree"}},
		},
		{
			name:      "group option without group output",
			output:    ast.Output{Name: "interleaved", Group: ast.OutputGroup{Begin: "begin"}},
			wantError: "--output-group-begin without --output=group",
		},
		{
			name:      "TUI status without TUI output",
			output:    ast.Output{Name: "interleaved", TUI: ast.OutputTUI{Status: "labels"}},
			wantError: "--output-tui-status without --output=tui",
		},
		{
			name:      "TUI navigator without TUI output",
			output:    ast.Output{Name: "interleaved", TUI: ast.OutputTUI{TaskNavigator: "tree"}},
			wantError: "--output-tui-task-navigator without --output=tui",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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
