package filepathext

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSmartJoinRelativePathContainingSpecialVariableName(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	for _, variable := range []string{".ROOT_DIR", ".TASKFILE_DIR", ".USER_WORKING_DIR"} {
		t.Run(variable, func(t *testing.T) {
			t.Parallel()
			for _, relative := range []string{
				filepath.Join("project"+variable, "file.txt"),
				filepath.Join(variable, "file.txt"),
				filepath.Join("{{.PROJECT}}"+variable, "file.txt"),
			} {
				require.False(t, IsAbs(relative), relative)
				require.Equal(t, filepath.Join(base, relative), SmartJoin(base, relative))
			}
		})
	}
}

func TestSmartJoinAbsoluteAndTemplatePaths(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	absolute := filepath.Join(t.TempDir(), "file.txt")
	for _, path := range []string{
		absolute,
		"{{.ROOT_DIR}}/file.txt",
		"{{ .TASKFILE_DIR }}/file.txt",
		"{{- .USER_WORKING_DIR -}}/file.txt",
		"{{.ROOT_DIR | toSlash}}/file.txt",
		"{{.PROJECT}}/{{.ROOT_DIR}}/file.txt",
	} {
		require.True(t, IsAbs(path), path)
		require.Equal(t, path, SmartJoin(base, path))
	}
}
