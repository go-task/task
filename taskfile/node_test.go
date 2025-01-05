package taskfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScheme(t *testing.T) {
	t.Parallel()

	scheme, err := getScheme("https://github.com/foo/bar.git")
	assert.NoError(t, err)
	assert.Equal(t, "git", scheme)
	scheme, err = getScheme("https://github.com/foo/bar.git?ref=v1//taskfile/common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "git", scheme)
	scheme, err = getScheme("git@github.com:foo/bar.git?ref=main//Taskfile.yml")
	assert.NoError(t, err)
	assert.Equal(t, "git", scheme)
	scheme, err = getScheme("https://github.com/foo/common.yml")
	assert.NoError(t, err)
	assert.Equal(t, "https", scheme)
}
