package templater

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestReplaceGlobsKeepsUnresolvedDynamicVariables(t *testing.T) {
	t.Parallel()

	dynamic := "echo parent"
	vars := ast.NewVars()
	vars.Set("PARENT", ast.Var{Sh: &dynamic})
	vars.Set("STATIC", ast.Var{Value: "parent"})

	globs := ReplaceGlobs([]*ast.Glob{
		{Glob: "{{.PARENT}}/**/*"},
		{Glob: "{{.STATIC}}/**/*"},
	}, &Cache{Vars: vars})

	require.Len(t, globs, 2)
	// An unevaluated shell variable must keep its placeholder. Replacing it with
	// an empty string produces "/**/*", which makes the globber traverse the
	// filesystem root and can hang on symlink cycles.
	require.Equal(t, "{{.PARENT}}/**/*", globs[0].Glob)
	require.Equal(t, "parent/**/*", globs[1].Glob)
}
