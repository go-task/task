package filepathext

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAbs(t *testing.T) {
	t.Parallel()

	absolute, err := filepath.Abs("file.txt")
	require.NoError(t, err)

	for _, test := range []struct {
		path    string
		wantAbs bool
	}{
		{"", false},
		{"project.ROOT_DIR/file.txt", false},
		{".TASKFILE_DIR/file.txt", false},
		{"project.USER_WORKING_DIR/file.txt", false},
		{"{{.PROJECT}}.ROOT_DIR{{.SUFFIX}}/file.txt", false},
		{"{{ .ROOT_DIR/file.txt", false},
		{"{{.PROJECT}}/file.txt", false},
		{"{{.ROOT_DIR_SUFFIX}}/file.txt", false},
		{"{{.ROOT_DIR2}}/file.txt", false},
		{"{{.ROOT_DIRsuffix}}/file.txt", false},
		{"{{.ROOT_DIR变量}}/file.txt", false},
		{absolute, true},
		{"{{.ROOT_DIR}}/file.txt", true},
		{"{{ .TASKFILE_DIR }}/file.txt", true},
		{"{{- .USER_WORKING_DIR -}}/file.txt", true},
		{"{{.ROOT_DIR|toSlash}}/file.txt", true},
		{"{{(.ROOT_DIR)}}/file.txt", true},
		{"{{.PROJECT}}/{{.ROOT_DIR}}/file.txt", true},
		{"{{\n.TASKFILE_DIR\n}}/file.txt", true},
		{`{{joinPath .ROOT_DIR "src"}}/file.txt`, true},
	} {
		t.Run(test.path, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.wantAbs, IsAbs(test.path))
		})
	}
}
