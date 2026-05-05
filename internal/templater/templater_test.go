package templater

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestTemplateFuncs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "joinPath",
			template: `{{ joinPath  .BaseDir "dir" "file.txt" }}`,
			expected: func() string {
				switch os := runtime.GOOS; os {
				case "windows":
					return "base\\dir\\file.txt"
				default:
					return "base/dir/file.txt"
				}
			}(),
		},
		{
			name:     "joinPath with single argument",
			template: `{{ joinPath "dir1" }}`,
			expected: "dir1",
		},
		{
			name:     "joinPathList",
			template: `{{ joinPathList .BaseDir "subdir" "file.txt" }}`,
			expected: func() string {
				switch os := runtime.GOOS; os {
				case "windows":
					return "base;subdir;file.txt"
				default:
					return "base:subdir:file.txt"
				}
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			vars := ast.NewVars(
				&ast.VarElement{
					Key:   "BaseDir",
					Value: ast.Var{Value: "base"},
				},
			)
			cache := &Cache{Vars: vars}
			cache.ResetCache()
			result := Replace(tc.template, cache)
			require.NoError(t, cache.Err())
			assert.Equal(t, tc.expected, result)
		})
	}
}
