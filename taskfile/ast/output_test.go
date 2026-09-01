package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestOutputTUIUnmarshalYAML(t *testing.T) {
	t.Parallel()

	var output Output
	require.NoError(t, yaml.Unmarshal([]byte("tui:\n  status: labels\n  task_navigator: tree\n"), &output))
	assert.Equal(t, "tui", output.Name)
	assert.Equal(t, "labels", output.TUI.Status)
	assert.Equal(t, "tree", output.TUI.TaskNavigator)
}

func TestOutputMappingRejectsMultipleStyles(t *testing.T) {
	t.Parallel()

	var output Output
	err := yaml.Unmarshal([]byte("group: {}\ntui: {}\n"), &output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one")
}
