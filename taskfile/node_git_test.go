package taskfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitNode_ssh(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("git@github.com:foo/bar.git//Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "Taskfile.yml", node.filepath)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//Taskfile.yml?ref=main", node.fullURL)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git", node.baseURL)
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//common.yml?ref=main", entrypoint)
}

func TestGitNode_sshWithDir(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("git@github.com:foo/bar.git//directory/Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "directory/Taskfile.yml", node.filepath)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//directory/Taskfile.yml?ref=main", node.fullURL)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git", node.baseURL)
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git//directory/common.yml?ref=main", entrypoint)
}

func TestGitNode_https(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("https://git:token@github.com/foo/bar.git//Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "Taskfile.yml", node.filepath)
	assert.Equal(t, "https://git:xxxxx@github.com/foo/bar.git//Taskfile.yml?ref=main", node.fullURL)
	assert.Equal(t, "https://git:token@github.com/foo/bar.git", node.baseURL)
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "https://git:token@github.com/foo/bar.git//common.yml?ref=main", entrypoint)
}

func TestGitNode_httpsWithDir(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("https://git:token@github.com/foo/bar.git//directory/Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "directory/Taskfile.yml", node.filepath)
	assert.Equal(t, "https://git:xxxxx@github.com/foo/bar.git//directory/Taskfile.yml?ref=main", node.fullURL)
	assert.Equal(t, "https://git:token@github.com/foo/bar.git", node.baseURL)
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "https://git:token@github.com/foo/bar.git//directory/common.yml?ref=main", entrypoint)
}

func TestGitNode_FilenameAndDir(t *testing.T) {
	t.Parallel()

	node, err := NewGitNode("https://github.com/foo/bar.git//directory/Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	filename, dir := node.FilenameAndLastDir()
	assert.Equal(t, "Taskfile.yml", filename)
	assert.Equal(t, "directory", dir)

	node, err = NewGitNode("https://github.com/foo/bar.git//Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	filename, dir = node.FilenameAndLastDir()
	assert.Equal(t, "Taskfile.yml", filename)
	assert.Equal(t, ".", dir)

	node, err = NewGitNode("https://github.com/foo/bar.git//multiple/directory/Taskfile.yml?ref=main", "", false)
	assert.NoError(t, err)
	filename, dir = node.FilenameAndLastDir()
	assert.Equal(t, "Taskfile.yml", filename)
	assert.Equal(t, "directory", dir)
}
