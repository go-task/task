package taskfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPNode_CacheKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		entrypoint  string
		expectedKey string
	}{
		{
			entrypoint:  "https://github.com",
			expectedKey: "http.github.com..996e1f714b08e971ec79e3bea686287e66441f043177999a13dbc546d8fe402a",
		},
		{
			entrypoint:  "https://github.com/Taskfile.yml",
			expectedKey: "http.github.com.Taskfile.yml.85b3c3ad71b78dc74e404c7b4390fc13672925cb644a4d26c21b9f97c17b5fc0",
		},
		{
			entrypoint:  "https://github.com/foo",
			expectedKey: "http.github.com.foo.df3158dafc823e6847d9bcaf79328446c4877405e79b100723fa6fd545ed3e2b",
		},
		{
			entrypoint:  "https://github.com/foo/Taskfile.yml",
			expectedKey: "http.github.com.foo.Taskfile.yml.aea946ea7eb6f6bb4e159e8b840b6b50975927778b2e666df988c03bbf10c4c4",
		},
		{
			entrypoint:  "https://github.com/foo/bar",
			expectedKey: "http.github.com.foo.bar.d3514ad1d4daedf9cc2825225070b49ebc8db47fa5177951b2a5b9994597570c",
		},
		{
			entrypoint:  "https://github.com/foo/bar/Taskfile.yml",
			expectedKey: "http.github.com.bar.Taskfile.yml.b9cf01e01e47c0e96ea536e1a8bd7b3a6f6c1f1881bad438990d2bfd4ccd0ac0",
		},
	}

	for _, tt := range tests {
		node, err := NewHTTPNode(tt.entrypoint, "", false)
		require.NoError(t, err)
		key := node.CacheKey()
		assert.Equal(t, tt.expectedKey, key)
	}
}
