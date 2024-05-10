package taskfile

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsGitNode(t *testing.T) {
	assert.True(t, isGitNode("https://github.com/foo/bar.git"))
	assert.True(t, isGitNode("https://github.com/foo/bar.git?ref=v1//taskfile/common.yml"))
	assert.True(t, isGitNode("git@github.com/foo/bar?ref=main//Taskfile.yml"))
}
