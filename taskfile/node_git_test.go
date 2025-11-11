package taskfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitNode_ssh(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("git@github.com:foo/bar.git//Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "Taskfile.yml", node.path)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//Taskfile.yml?ref=main", node.Location())
	assert.Equal(t, "ssh://git@github.com/foo/bar.git", node.url.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//common.yml?ref=main", entrypoint)
}

func TestGitNode_sshWithAltRepo(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("git@github.com:foo/bar.git//Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)

	entrypoint, err := node.ResolveEntrypoint("git@github.com:foo/other.git//Taskfile.yml?ref=dev")
	assert.NoError(t, err)
	assert.Equal(t, "git@github.com:foo/other.git//Taskfile.yml?ref=dev", entrypoint)
}

func TestGitNode_sshWithDir(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("git@github.com:foo/bar.git//directory/Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "directory/Taskfile.yml", node.path)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//directory/Taskfile.yml?ref=main", node.Location())
	assert.Equal(t, "ssh://git@github.com/foo/bar.git", node.url.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//directory/common.yml?ref=main", entrypoint)
}

func TestGitNode_https(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("https://github.com/foo/bar.git//Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "Taskfile.yml", node.path)
	assert.Equal(t, "https://github.com/foo/bar.git//Taskfile.yml?ref=main", node.Location())
	assert.Equal(t, "https://github.com/foo/bar.git", node.url.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/foo/bar.git//common.yml?ref=main", entrypoint)
}

func TestGitNode_httpsWithDir(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("https://github.com/foo/bar.git//directory/Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "directory/Taskfile.yml", node.path)
	assert.Equal(t, "https://github.com/foo/bar.git//directory/Taskfile.yml?ref=main", node.Location())
	assert.Equal(t, "https://github.com/foo/bar.git", node.url.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/foo/bar.git//directory/common.yml?ref=main", entrypoint)
}

func TestGitNode_CacheKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		entrypoint  string
		expectedKey string
	}{
		{
			entrypoint:  "https://github.com/foo/bar.git//directory/Taskfile.yml?ref=main",
			expectedKey: "git.github.com.directory.Taskfile.yml.f1ddddac425a538870230a3e38fc0cded4ec5da250797b6cab62c82477718fbb",
		},
		{
			entrypoint:  "https://github.com/foo/bar.git//Taskfile.yml?ref=main",
			expectedKey: "git.github.com.Taskfile.yml.39d28c1ff36f973705ae188b991258bbabaffd6d60bcdde9693d157d00d5e3a4",
		},
		{
			entrypoint:  "https://github.com/foo/bar.git//multiple/directory/Taskfile.yml?ref=main",
			expectedKey: "git.github.com.directory.Taskfile.yml.1b6d145e01406dcc6c0aa572e5a5d1333be1ccf2cae96d18296d725d86197d31",
		},
	}

	for _, tt := range tests {
		node, err := NewGitNode(tt.entrypoint, "", false)
		require.NoError(t, err)
		key := node.CacheKey()
		assert.Equal(t, tt.expectedKey, key)
	}
}

func TestGitNode_buildURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		entrypoint  string
		expectedURL string
	}{
		{
			name:        "HTTPS with ref",
			entrypoint:  "https://github.com/foo/bar.git//Taskfile.yml?ref=main",
			expectedURL: "git::https://github.com/foo/bar.git?ref=main&depth=1",
		},
		{
			name:        "SSH with ref",
			entrypoint:  "git@github.com:foo/bar.git//Taskfile.yml?ref=main",
			expectedURL: "git::ssh://git@github.com/foo/bar.git?ref=main&depth=1",
		},
		{
			name:        "HTTPS with tag ref",
			entrypoint:  "https://github.com/foo/bar.git//Taskfile.yml?ref=v1.0.0",
			expectedURL: "git::https://github.com/foo/bar.git?ref=v1.0.0&depth=1",
		},
		{
			name:        "HTTPS without ref (uses remote HEAD)",
			entrypoint:  "https://github.com/foo/bar.git//Taskfile.yml",
			expectedURL: "git::https://github.com/foo/bar.git?ref=HEAD&depth=1",
		},
		{
			name:        "SSH with directory path",
			entrypoint:  "git@github.com:foo/bar.git//directory/Taskfile.yml?ref=dev",
			expectedURL: "git::ssh://git@github.com/foo/bar.git?ref=dev&depth=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node, err := NewGitNode(tt.entrypoint, "", false)
			require.NoError(t, err)
			gotURL := node.buildURL()
			assert.Equal(t, tt.expectedURL, gotURL)
		})
	}
}
