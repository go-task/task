package taskfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdinNodeReadPreservesInput(t *testing.T) { //nolint:paralleltest // replaces process-wide stdin
	node, err := NewStdinNode("")
	require.NoError(t, err)

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "line longer than 64 KiB",
			input: []byte(strings.Repeat("a", 64*1024+1)),
		},
		{
			name:  "no trailing newline",
			input: []byte("version: '3'"),
		},
		{
			name:  "CRLF line endings",
			input: []byte("version: '3'\r\ntasks:\r\n"),
		},
		{
			name:  "empty input",
			input: []byte{},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests replace process-wide stdin
		t.Run(tt.name, func(t *testing.T) {
			replaceStdin(t, tt.input)

			got, err := node.Read()
			require.NoError(t, err)
			assert.Equal(t, tt.input, got)
		})
	}
}

func replaceStdin(t *testing.T, input []byte) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "stdin")
	require.NoError(t, os.WriteFile(path, input, 0o600))

	stdin, err := os.Open(path)
	require.NoError(t, err)

	original := os.Stdin
	os.Stdin = stdin
	t.Cleanup(func() {
		os.Stdin = original
		require.NoError(t, stdin.Close())
	})
}
