package taskfile

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGitNode(t *testing.T) {
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

func TestGitNode_WithoutRef(t *testing.T) {
	node, err := NewGitNode("git@github.com/foo/bar?ref=main//Taskfile.yml", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "Taskfile.yml", node.path)
	assert.Equal(t, "github.com/foo/bar?ref=main//Taskfile.yml", node.rawUrl)
	assert.Equal(t, "github.com/foo/bar", node.URL.String())
}

func TestGitNode_WithoutPath(t *testing.T) {
	node, err := NewGitNode("git@github.com/foo/bar?ref=main//Taskfile.yml", "", false)
	assert.NoError(t, err)
	assert.Equal(t, "main", node.ref)
	assert.Equal(t, "Taskfile.yml", node.path)
	assert.Equal(t, "github.com/foo/bar?ref=main//Taskfile.yml", node.rawUrl)
	assert.Equal(t, "github.com/foo/bar", node.URL.String())
}
