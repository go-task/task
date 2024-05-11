package taskfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGitNode(t *testing.T) {
	isGit, err := isGitNode("https://github.com/foo/bar.git")
	assert.NoError(t, err)
	assert.True(t, isGit)
	isGit, err = isGitNode("https://github.com/foo/bar.git?ref=v1//taskfile/common.yml")
	assert.NoError(t, err)
	assert.True(t, isGit)
	isGit, err = isGitNode("git@github.com:foo/bar.git?ref=main//Taskfile.yml")
	assert.NoError(t, err)
	assert.True(t, isGit)
}

func TestIsNotGitNode(t *testing.T) {
	isGit, err := isGitNode("https://github.com/foo/common.yml")
	assert.NoError(t, err)
	assert.False(t, isGit)
}
