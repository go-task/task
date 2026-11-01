package refs_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/refs"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestResolveEnum(t *testing.T) {
	t.Parallel()

	vars := ast.NewVars()
	vars.Set("ALLOWED_ENVS", ast.Var{Value: []any{"dev", "staging", "prod"}})

	t.Run("resolves a static ref into values", func(t *testing.T) {
		t.Parallel()

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Ref: ".ALLOWED_ENVS"}}

		resolved := refs.ResolveEnum(v, vars)

		require.Equal(t, []string{"dev", "staging", "prod"}, resolved.Enum.Value)
		require.Empty(t, v.Enum.Value, "input var must not be mutated")
		require.Equal(t, ".ALLOWED_ENVS", v.Enum.Ref)
	})

	t.Run("leaves an unresolvable ref as-is", func(t *testing.T) {
		t.Parallel()

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Ref: ".NONEXISTENT"}}

		require.Empty(t, refs.ResolveEnum(v, vars).Enum.Value)
	})

	t.Run("passes through a static enum unchanged", func(t *testing.T) {
		t.Parallel()

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Value: []string{"a", "b"}}}

		require.Same(t, v, refs.ResolveEnum(v, vars))
	})

	t.Run("accepts the list types template functions return", func(t *testing.T) {
		t.Parallel()

		vars := ast.NewVars()
		vars.Set("MAP", ast.Var{Value: map[string]any{"dev": 1, "prod": 2}})

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Ref: "keys .MAP | sortAlpha"}}

		require.Equal(t, []string{"dev", "prod"}, refs.ResolveEnum(v, vars).Enum.Value)
	})
}
