package task

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

func newTestPromptExecutor(t *testing.T) *Executor {
	t.Helper()
	e := NewExecutor(WithAssumeTerm(true))
	e.Compiler = &Compiler{
		Dir:          t.TempDir(),
		TaskfileEnv:  ast.NewVars(),
		TaskfileVars: ast.NewVars(),
		Logger:       &logger.Logger{Stderr: io.Discard},
	}
	return e
}

func TestResolveEnumRefForPrompt(t *testing.T) {
	t.Parallel()

	e := newTestPromptExecutor(t)
	dir := e.Compiler.Dir

	vars := ast.NewVars()
	vars.Set("ALLOWED_ENVS", ast.Var{Value: []any{"dev", "staging", "prod"}})

	t.Run("resolves a static ref into values", func(t *testing.T) {
		t.Parallel()

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Ref: ".ALLOWED_ENVS"}}

		resolved := e.resolveEnumRefForPrompt(v, vars, dir)

		require.Equal(t, []string{"dev", "staging", "prod"}, getEnumValues(resolved.Enum))
		require.Empty(t, v.Enum.Value, "input var must not be mutated")
		require.Equal(t, ".ALLOWED_ENVS", v.Enum.Ref)
	})

	t.Run("leaves an unresolvable ref as-is", func(t *testing.T) {
		t.Parallel()

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Ref: ".NONEXISTENT"}}

		require.Empty(t, getEnumValues(e.resolveEnumRefForPrompt(v, vars, dir).Enum))
	})

	t.Run("passes through a static enum unchanged", func(t *testing.T) {
		t.Parallel()

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Value: []string{"a", "b"}}}

		require.Same(t, v, e.resolveEnumRefForPrompt(v, vars, dir))
	})

	t.Run("resolves a ref to a dynamic sh variable", func(t *testing.T) {
		t.Parallel()

		dynamicVars := ast.NewVars()
		dynamicVars.Set("AVAILABLE_SERVICES", ast.Var{Sh: strPtr("printf 'api\nweb\ndb\n'")})

		v := &ast.VarsWithValidation{Name: "SERVICE", Enum: &ast.Enum{Ref: ".AVAILABLE_SERVICES | splitLines | compact"}}

		resolved := e.resolveEnumRefForPrompt(v, dynamicVars, dir)

		require.Equal(t, []string{"api", "web", "db"}, getEnumValues(resolved.Enum))
		require.Empty(t, v.Enum.Value, "input var must not be mutated")
	})

	t.Run("keeps free-form fallback when a dynamic ref is empty", func(t *testing.T) {
		t.Parallel()

		dynamicVars := ast.NewVars()
		dynamicVars.Set("AVAILABLE_SERVICES", ast.Var{Sh: strPtr("printf ''")})

		v := &ast.VarsWithValidation{Name: "SERVICE", Enum: &ast.Enum{Ref: ".AVAILABLE_SERVICES | splitLines | compact"}}

		require.Empty(t, getEnumValues(e.resolveEnumRefForPrompt(v, dynamicVars, dir).Enum))
	})

	t.Run("resolves a dynamic ref even when an unrelated sh var fails", func(t *testing.T) {
		t.Parallel()

		dynamicVars := ast.NewVars()
		dynamicVars.Set("BROKEN", ast.Var{Sh: strPtr("exit 1")})
		dynamicVars.Set("AVAILABLE_SERVICES", ast.Var{Sh: strPtr("printf 'api\nweb\ndb\n'")})

		v := &ast.VarsWithValidation{Name: "SERVICE", Enum: &ast.Enum{Ref: ".AVAILABLE_SERVICES | splitLines | compact"}}

		resolved := e.resolveEnumRefForPrompt(v, dynamicVars, dir)

		require.Equal(t, []string{"api", "web", "db"}, getEnumValues(resolved.Enum))
	})
}

func strPtr(s string) *string {
	return &s
}
