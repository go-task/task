package fsext

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFastRecursiveGlob(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "root")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "nested", "deeper"), 0o755))

	files := []string{
		filepath.Join(root, "direct.yaml"),
		filepath.Join(root, "nested", "alpha.yaml"),
		filepath.Join(root, "nested", "deeper", "beta.yaml"),
		filepath.Join(root, "nested", "ignored.txt"),
	}
	for _, file := range files {
		require.NoError(t, os.WriteFile(file, nil, 0o600))
	}

	got, ok, err := FastRecursiveGlob(filepath.Join(root, "**", "*.yaml"))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{
		filepath.ToSlash(files[0]),
		filepath.ToSlash(files[1]),
		filepath.ToSlash(files[2]),
	}, got)

	got, ok, err = FastRecursiveGlob(filepath.Join(root, "**", "deeper", "*.yaml"))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{filepath.ToSlash(files[2])}, got)
}

func TestFastRecursiveGlobFallback(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "root")
	tests := []struct {
		name    string
		pattern string
	}{
		{name: "no recursive marker", pattern: filepath.Join(root, "*.yaml")},
		{name: "wildcard root", pattern: filepath.Join(root, "*", "**", "*.yaml")},
		{name: "brace suffix", pattern: filepath.Join(root, "**", "*.{yaml,yml}")},
		{name: "multiple recursive markers", pattern: filepath.Join(root, "**", "nested", "**", "*.yaml")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok, err := FastRecursiveGlob(tt.pattern)
			require.NoError(t, err)
			require.False(t, ok)
			require.Nil(t, got)
		})
	}
}

func TestFastRecursiveGlobNonDirectoryRoot(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "root.yaml")
	require.NoError(t, os.WriteFile(root, nil, 0o600))

	got, ok, err := FastRecursiveGlob(filepath.Join(root, "**", "*.yaml"))
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, got)
}

func TestFastRecursiveGlobSymlinkDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "root")
	target := filepath.Join(tempDir, "target")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.MkdirAll(target, 0o755))
	file := filepath.Join(target, "file.yaml")
	require.NoError(t, os.WriteFile(file, nil, 0o600))
	if err := os.Symlink(target, filepath.Join(root, "linked")); err != nil {
		t.Skipf("cannot create directory symlink: %v", err)
	}

	got, ok, err := FastRecursiveGlob(filepath.Join(root, "**", "*.yaml"))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{filepath.ToSlash(filepath.Join(root, "linked", "file.yaml"))}, got)
}

func TestFastRecursiveGlobSymlinkLoop(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "root")
	require.NoError(t, os.MkdirAll(root, 0o755))
	file := filepath.Join(root, "file.yaml")
	require.NoError(t, os.WriteFile(file, nil, 0o600))
	if err := os.Symlink(root, filepath.Join(root, "loop")); err != nil {
		t.Skipf("cannot create directory symlink: %v", err)
	}

	got, ok, err := FastRecursiveGlob(filepath.Join(root, "**", "*.yaml"))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{filepath.ToSlash(file)}, got)
}
