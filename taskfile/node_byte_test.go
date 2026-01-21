package taskfile

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed testdata/node_byte_taskfile.yaml
var taskfileYamlBytes []byte

func TestByteNode(t *testing.T) {
	t.Parallel()
	workingDir := t.TempDir()

	node, err := NewByteNode(taskfileYamlBytes, workingDir)
	assert.NoError(t, err)
	assert.Equal(t, "__bytes__", node.Location())
	assert.Equal(t, workingDir, node.Dir())
	assert.True(t, node.Remote())
	data, err := node.Read()
	assert.NoError(t, err)
	assert.Equal(t, taskfileYamlBytes, data)
}
