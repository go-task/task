package filepathext

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAbs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "plain relative path",
			path: filepath.Join("foo", "bar"),
			want: false,
		},
		{
			name: "absolute path",
			path: filepath.Join(string(filepath.Separator), "foo", "bar"),
			want: true,
		},
		{
			name: "special dir template expression",
			path: "{{.ROOT_DIR}}/foo",
			want: true,
		},
		{
			name: "taskfile dir template expression",
			path: "{{.TASKFILE_DIR}}/foo",
			want: true,
		},
		{
			name: "user working dir template expression",
			path: "{{.USER_WORKING_DIR}}/foo",
			want: true,
		},
		{
			// Regression test for https://github.com/go-task/task/issues/3021.
			// An ordinary relative path whose name merely contains a special
			// variable name must NOT be treated as absolute.
			name: "relative path containing root dir substring",
			path: filepath.Join("project.ROOT_DIR", "file.txt"),
			want: false,
		},
		{
			name: "relative path containing taskfile dir substring",
			path: filepath.Join("project.TASKFILE_DIR", "file.txt"),
			want: false,
		},
		{
			name: "relative path containing user working dir substring",
			path: filepath.Join("project.USER_WORKING_DIR", "file.txt"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsAbs(tt.path))
		})
	}
}

func TestSmartJoin(t *testing.T) {
	t.Parallel()

	base := filepath.Join("base", "dir")

	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{
			name: "joins relative path",
			a:    base,
			b:    filepath.Join("foo", "bar"),
			want: filepath.Join(base, "foo", "bar"),
		},
		{
			name: "returns absolute path unchanged",
			a:    base,
			b:    filepath.Join(string(filepath.Separator), "abs", "path"),
			want: filepath.Join(string(filepath.Separator), "abs", "path"),
		},
		{
			name: "returns special dir template path unchanged",
			a:    base,
			b:    "{{.ROOT_DIR}}/foo",
			want: "{{.ROOT_DIR}}/foo",
		},
		{
			// Regression test for https://github.com/go-task/task/issues/3021.
			name: "joins relative path containing special variable name",
			a:    base,
			b:    filepath.Join("project.ROOT_DIR", "file.txt"),
			want: filepath.Join(base, "project.ROOT_DIR", "file.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, SmartJoin(tt.a, tt.b))
		})
	}
}
