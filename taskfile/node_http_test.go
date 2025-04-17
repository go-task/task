package taskfile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPNode_https(t *testing.T) {
	t.Parallel()

	node, err := NewHTTPNode("https://raw.githubusercontent.com/my-org/my-repo/main/Taskfile.yml", "", false, time.Second)
	require.NoError(t, err)
	assert.Equal(t, time.Second, node.timeout)
	assert.Equal(t, "https://raw.githubusercontent.com/my-org/my-repo/main/Taskfile.yml", node.url.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	require.NoError(t, err)
	assert.Equal(t, "https://raw.githubusercontent.com/my-org/my-repo/main/common.yml", entrypoint)
}

func TestHTTPNode_redaction(t *testing.T) {
	t.Parallel()

	node, err := NewHTTPNode("https://user:password@example.com/Taskfile.yml", "", false, time.Second)

	t.Run("the location is redacted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, err)
		assert.Equal(t, "https://user:xxxxx@example.com/Taskfile.yml", node.Location())
	})

	t.Run("resolved entrypoints contain the username and password", func(t *testing.T) {
		t.Parallel()
		location, err := node.ResolveEntrypoint("common.yaml")
		require.NoError(t, err)
		assert.Equal(t, "https://user:password@example.com/common.yaml", location)
	})
}

func TestHTTPNode_FilenameAndDir(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		entrypoint string
		filename   string
		dir        string
	}{
		"file at root": {
			entrypoint: "https://example.com/Taskfile.yaml",
			filename:   "Taskfile.yaml",
			dir:        ".",
		},
		"file in folder": {
			entrypoint: "https://example.com/taskfiles/Taskfile.yaml",
			filename:   "Taskfile.yaml",
			dir:        "taskfiles",
		},
		"nested structure": {
			entrypoint: "https://raw.githubusercontent.com/my-org/my-repo/main/Taskfile.yaml",
			filename:   "Taskfile.yaml",
			dir:        "main",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			node, err := NewHTTPNode(tt.entrypoint, "", false, time.Second)
			require.NoError(t, err)
			dir, filename := node.FilenameAndLastDir()
			assert.Equal(t, tt.filename, filename)
			assert.Equal(t, tt.dir, dir)
		})
	}
}
