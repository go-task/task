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
	require.NoError(t, yaml.Unmarshal([]byte("tui:\n  hide_internal: true\n  status: labels\n"), &output))
	assert.Equal(t, "tui", output.Name)
	assert.True(t, output.TUI.HideInternal)
	assert.Equal(t, "labels", output.TUI.Status)
}

func TestOutputMappingRejectsMultipleStyles(t *testing.T) {
	t.Parallel()

	var output Output
	err := yaml.Unmarshal([]byte("group: {}\ntui: {}\n"), &output)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one")
}
