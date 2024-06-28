package taskfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitNode_ssh(t *testing.T) {
	node, err := NewGitNode("git@github.com:foo/bar.git?ref=main//Taskfile.yml", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "Taskfile.yml", node.path)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git?ref=main//Taskfile.yml", node.rawUrl)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git", node.URL.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git?ref=main//common.yml", entrypoint)
}

func TestGitNode_sshWithDir(t *testing.T) {
	node, err := NewGitNode("git@github.com:foo/bar.git?ref=main//directory/Taskfile.yml", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "directory/Taskfile.yml", node.path)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git?ref=main//directory/Taskfile.yml", node.rawUrl)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git", node.URL.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "ssh://git@github.com/foo/bar.git?ref=main//directory/common.yml", entrypoint)
}

func TestGitNode_https(t *testing.T) {
	node, err := NewGitNode("https://github.com/foo/bar.git?ref=main//Taskfile.yml", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "Taskfile.yml", node.path)
	assert.Equal(t, "https://github.com/foo/bar.git?ref=main//Taskfile.yml", node.rawUrl)
	assert.Equal(t, "https://github.com/foo/bar.git", node.URL.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/foo/bar.git?ref=main//common.yml", entrypoint)
}

func TestGitNode_httpsWithDir(t *testing.T) {
	node, err := NewGitNode("https://github.com/foo/bar.git?ref=main//directory/Taskfile.yml", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "directory/Taskfile.yml", node.path)
	assert.Equal(t, "https://github.com/foo/bar.git?ref=main//directory/Taskfile.yml", node.rawUrl)
	assert.Equal(t, "https://github.com/foo/bar.git", node.URL.String())
	entrypoint, err := node.ResolveEntrypoint("common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/foo/bar.git?ref=main//directory/common.yml", entrypoint)
}

func TestGitNode_FilenameAndDir(t *testing.T) {
	node, err := NewGitNode("https://github.com/foo/bar.git?ref=main//directory/Taskfile.yml", "", false)
	assert.NoError(t, err)
	filename, dir := node.FilenameAndLastDir()
	assert.Equal(t, "Taskfile.yml", filename)
	assert.Equal(t, "directory", dir)

	node, err = NewGitNode("https://github.com/foo/bar.git?ref=main//Taskfile.yml", "", false)
	assert.NoError(t, err)
	filename, dir = node.FilenameAndLastDir()
	assert.Equal(t, "Taskfile.yml", filename)
	assert.Equal(t, ".", dir)

	node, err = NewGitNode("https://github.com/foo/bar.git?ref=main//multiple/directory/Taskfile.yml", "", false)
	assert.NoError(t, err)
	filename, dir = node.FilenameAndLastDir()
	assert.Equal(t, "Taskfile.yml", filename)
	assert.Equal(t, "directory", dir)
}
