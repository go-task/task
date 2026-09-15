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
		name         string
		enabled      bool
		statusSet    bool
		navigatorSet bool
		wantError    string
	}{
		{name: "TUI without options", enabled: true},
		{name: "TUI with options", enabled: true, statusSet: true, navigatorSet: true},
		{name: "status flag without TUI", statusSet: true, wantError: "--tui-status without --tui"},
		{name: "navigator flag without TUI", navigatorSet: true, wantError: "--tui-task-navigator without --tui"},
		// A .taskrc.yml default leaves the flags unchanged, so an ordinary run
		// is not failed by settings that only apply to the interface.
		{name: "options configured, TUI off"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := validateTUIOptions(test.enabled, test.statusSet, test.navigatorSet)
			if test.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantError)
		})
	}
}

func TestValidateTUIPrompting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		tui            bool
		interactive    bool
		interactiveSet bool
		wantError      string
	}{
		{name: "TUI alone"},
		{name: "TUI without the flag", tui: true},
		{name: "TUI with prompting asked for", tui: true, interactive: true, interactiveSet: true},
		{
			name: "TUI with prompting turned off", tui: true, interactiveSet: true,
			wantError: "--interactive=false with --tui",
		},
		// The flag defaults to false, so an unset flag must not be mistaken for
		// a request to turn prompting off.
		{name: "TUI with the flag left alone", tui: true, interactive: false},
		// Without the TUI the flag means what it always has.
		{name: "prompting turned off on its own", interactiveSet: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := validateTUIPrompting(test.tui, test.interactive, test.interactiveSet)
			if test.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantError)
		})
	}
}
