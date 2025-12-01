package templater

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestReplaceVarWithExtra_ResolvesRefFromExtra(t *testing.T) {
	t.Parallel()

	cache := Cache{Vars: ast.NewVars()}
	extra := map[string]any{"ITEM": "a"}

	got := ReplaceVarWithExtra(ast.Var{Ref: ".ITEM"}, &cache, extra)

	require.Equal(t, "a", got.Value)
}
