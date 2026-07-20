package task

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestResolveEnumRefForPrompt(t *testing.T) {
	t.Parallel()

	vars := ast.NewVars()
	vars.Set("ALLOWED_ENVS", ast.Var{Value: []any{"dev", "staging", "prod"}})

	t.Run("resolves a static ref into values", func(t *testing.T) {
		t.Parallel()

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Ref: ".ALLOWED_ENVS"}}

		resolved := resolveEnumRefForPrompt(v, vars)

		require.Equal(t, []string{"dev", "staging", "prod"}, getEnumValues(resolved.Enum))
		require.Empty(t, v.Enum.Value, "input var must not be mutated")
		require.Equal(t, ".ALLOWED_ENVS", v.Enum.Ref)
	})

	t.Run("leaves an unresolvable ref as-is", func(t *testing.T) {
		t.Parallel()

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Ref: ".NONEXISTENT"}}

		require.Empty(t, getEnumValues(resolveEnumRefForPrompt(v, vars).Enum))
	})

	t.Run("passes through a static enum unchanged", func(t *testing.T) {
		t.Parallel()

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Value: []string{"a", "b"}}}

		require.Same(t, v, resolveEnumRefForPrompt(v, vars))
	})
}
