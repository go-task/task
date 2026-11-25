package execext

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: "hello", want: "hello"},
		{name: "posix path", input: "/tmp/some dir/x", want: "/tmp/some dir/x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ExpandLiteral(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExpandLiteralWindowsPath(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "windows" {
		t.Skip("windows path separators only exist on windows")
	}

	input := `C:\Users\RUNNER~1\some dir\x\001`
	want := filepath.ToSlash(input)
	got, err := ExpandLiteral(input)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
