package task

import (
	"testing"

	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
	"github.com/stretchr/testify/require"
)

func TestResolveDotenvVarsNestedTemplates(t *testing.T) {
	t.Parallel()

	base := ast.NewVars()
	base.Set("BASE_DIR", ast.Var{Value: "/home/user"})

	cache := &templater.Cache{Vars: base}

	dotenv := ast.NewVars()
	dotenv.Set("MID_DIR", ast.Var{Value: "{{.BASE_DIR}}/nested"})
	dotenv.Set("FINAL_DIR", ast.Var{Value: "{{.MID_DIR}}/deeper"})
	dotenv.Set("FULL_PATH", ast.Var{Value: "{{.FINAL_DIR}}/file.txt"})

	resolved := resolveDotenvVars(dotenv, cache)

	mid, ok := resolved.Get("MID_DIR")
	require.True(t, ok)
	require.Equal(t, "/home/user/nested", mid.Value)

	finalDir, ok := resolved.Get("FINAL_DIR")
	require.True(t, ok)
	require.Equal(t, "/home/user/nested/deeper", finalDir.Value)

	fullPath, ok := resolved.Get("FULL_PATH")
	require.True(t, ok)
	require.Equal(t, "/home/user/nested/deeper/file.txt", fullPath.Value)
}
