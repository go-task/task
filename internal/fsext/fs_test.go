package fsext

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultDir(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name       string
		entrypoint string
		dir        string
		expected   string
	}{
		{
			name:       "default to current working directory",
			entrypoint: "",
			dir:        "",
			expected:   wd,
		},
		{
			name:       "resolves relative dir path",
			entrypoint: "",
			dir:        "./dir",
			expected:   filepath.Join(wd, "dir"),
		},
		{
			name:       "return entrypoint if set",
			entrypoint: filepath.Join(wd, "entrypoint"),
			dir:        "",
			expected:   "",
		},
		{
			name:       "if entrypoint and dir are set",
			entrypoint: filepath.Join(wd, "entrypoint"),
			dir:        filepath.Join(wd, "dir"),
			expected:   filepath.Join(wd, "dir"),
		},
		{
			name:       "if entrypoint and dir are set and dir is relative",
			entrypoint: filepath.Join(wd, "entrypoint"),
			dir:        "./dir",
			expected:   filepath.Join(wd, "dir"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, DefaultDir(tt.entrypoint, tt.dir))
		})
	}
}

func TestSearch(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name               string
		entrypoint         string
		dir                string
		possibleFilenames  []string
		expectedEntrypoint string
	}{
		{
			name:               "find foo.txt using relative entrypoint",
			entrypoint:         "./testdata/foo.txt",
			possibleFilenames:  []string{"foo.txt"},
			expectedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
		},
		{
			name:               "find foo.txt using absolute entrypoint",
			entrypoint:         filepath.Join(wd, "testdata", "foo.txt"),
			possibleFilenames:  []string{"foo.txt"},
			expectedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
		},
		{
			name:               "find foo.txt using relative dir",
			dir:                "./testdata",
			possibleFilenames:  []string{"foo.txt"},
			expectedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
		},
		{
			name:               "find foo.txt using absolute dir",
			dir:                filepath.Join(wd, "testdata"),
			possibleFilenames:  []string{"foo.txt"},
			expectedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
		},
		{
			name:               "find foo.txt using relative dir and relative entrypoint",
			entrypoint:         "./testdata/foo.txt",
			dir:                "./testdata/some/other/dir",
			possibleFilenames:  []string{"foo.txt"},
			expectedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
		},
		{
			name:               "find fs.go using no entrypoint or dir",
			entrypoint:         "",
			dir:                "",
			possibleFilenames:  []string{"fs.go"},
			expectedEntrypoint: filepath.Join(wd, "fs.go"),
		},
		{
			name:               "find ../../Taskfile.yml using no entrypoint or dir by walking",
			entrypoint:         "",
			dir:                "",
			possibleFilenames:  []string{"Taskfile.yml"},
			expectedEntrypoint: filepath.Join(wd, "..", "..", "Taskfile.yml"),
		},
		{
			name:               "find foo.txt first if listed first in possible filenames",
			entrypoint:         "./testdata",
			possibleFilenames:  []string{"foo.txt", "bar.txt"},
			expectedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
		},
		{
			name:               "find bar.txt first if listed first in possible filenames",
			entrypoint:         "./testdata",
			possibleFilenames:  []string{"bar.txt", "foo.txt"},
			expectedEntrypoint: filepath.Join(wd, "testdata", "bar.txt"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entrypoint, err := Search(tt.entrypoint, tt.dir, tt.possibleFilenames)
			require.NoError(t, err)
			require.Equal(t, tt.expectedEntrypoint, entrypoint)
			require.NoError(t, err)
		})
	}
}

func TestResolveDir(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name               string
		entrypoint         string
		resolvedEntrypoint string
		dir                string
		expectedDir        string
	}{
		{
			name:               "find foo.txt using relative entrypoint",
			entrypoint:         "./testdata/foo.txt",
			resolvedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
			expectedDir:        filepath.Join(wd, "testdata"),
		},
		{
			name:               "find foo.txt using absolute entrypoint",
			entrypoint:         filepath.Join(wd, "testdata", "foo.txt"),
			resolvedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
			expectedDir:        filepath.Join(wd, "testdata"),
		},
		{
			name:               "find foo.txt using relative dir",
			resolvedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
			dir:                "./testdata",
			expectedDir:        filepath.Join(wd, "testdata"),
		},
		{
			name:               "find foo.txt using absolute dir",
			resolvedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
			dir:                filepath.Join(wd, "testdata"),
			expectedDir:        filepath.Join(wd, "testdata"),
		},
		{
			name:               "find foo.txt using relative dir and relative entrypoint",
			entrypoint:         "./testdata/foo.txt",
			resolvedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
			dir:                "./testdata/some/other/dir",
			expectedDir:        filepath.Join(wd, "testdata", "some", "other", "dir"),
		},
		{
			name:               "find fs.go using no entrypoint or dir",
			entrypoint:         "",
			resolvedEntrypoint: filepath.Join(wd, "fs.go"),
			dir:                "",
			expectedDir:        wd,
		},
		{
			name:               "find ../../Taskfile.yml using no entrypoint or dir by walking",
			entrypoint:         "",
			resolvedEntrypoint: filepath.Join(wd, "..", "..", "Taskfile.yml"),
			dir:                "",
			expectedDir:        filepath.Join(wd, "..", ".."),
		},
		{
			name:               "find foo.txt first if listed first in possible filenames",
			entrypoint:         "./testdata",
			resolvedEntrypoint: filepath.Join(wd, "testdata", "foo.txt"),
			expectedDir:        filepath.Join(wd, "testdata"),
		},
		{
			name:               "find bar.txt first if listed first in possible filenames",
			entrypoint:         "./testdata",
			resolvedEntrypoint: filepath.Join(wd, "testdata", "bar.txt"),
			expectedDir:        filepath.Join(wd, "testdata"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, err := ResolveDir(tt.entrypoint, tt.resolvedEntrypoint, tt.dir)
			require.NoError(t, err)
			require.Equal(t, tt.expectedDir, dir)
			require.NoError(t, err)
		})
	}
}
