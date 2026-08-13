package task

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

// newTestPromptExecutor returns an Executor whose Compiler is wired up so that
// dynamic (sh:) variables can be evaluated during enum ref resolution.
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

// TestPromptEnumRefResolution exercises the real prompt path: fast-compiled
// task vars are resolved through resolveEnumRefForPrompt, exactly as
// promptDepsVars does, against a real Taskfile fixture.
func TestPromptEnumRefResolution(t *testing.T) {
	t.Parallel()

	e := NewExecutor(WithDir("testdata/enum_ref_prompt"), WithAssumeTerm(true))
	require.NoError(t, e.Setup())

	tests := []struct {
		name     string
		task     string
		varName  string
		wantEnum []string
	}{
		{
			name:     "enum ref to a dynamic sh variable is evaluated",
			task:     "deploy",
			varName:  "SERVICE",
			wantEnum: []string{"api", "web", "db"},
		},
		{
			name:     "enum ref to a static list variable resolves",
			task:     "release",
			varName:  "ENV",
			wantEnum: []string{"dev", "staging", "prod"},
		},
		{
			name:    "enum ref to an empty dynamic variable yields no options",
			task:    "deploy-empty",
			varName: "SERVICE",
			// resolveEnumRefs always assigns a (possibly empty) slice, which is
			// what keeps the prompter on free-form input.
			wantEnum: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			compiledTask, err := e.FastCompiledTask(&Call{Task: tt.task})
			require.NoError(t, err)

			missing := getMissingRequiredVars(compiledTask)
			require.Len(t, missing, 1)
			require.Equal(t, tt.varName, missing[0].Name)

			resolved := e.resolveEnumRefForPrompt(missing[0], compiledTask.Vars, compiledTask.Dir)

			require.Equal(t, tt.wantEnum, getEnumValues(resolved.Enum))
			require.Empty(t, missing[0].Enum.Value, "input var must not be mutated")
		})
	}
}

func TestResolveEnumRefForPrompt(t *testing.T) {
	t.Parallel()

	e := newTestPromptExecutor(t)
	dir := e.Compiler.Dir

	t.Run("resolves a static ref into values", func(t *testing.T) {
		t.Parallel()

		vars := ast.NewVars()
		vars.Set("ALLOWED_ENVS", ast.Var{Value: []any{"dev", "staging", "prod"}})

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Ref: ".ALLOWED_ENVS"}}

		resolved := e.resolveEnumRefForPrompt(v, vars, dir)

		require.Equal(t, []string{"dev", "staging", "prod"}, getEnumValues(resolved.Enum))
		require.Empty(t, v.Enum.Value, "input var must not be mutated")
	})

	t.Run("leaves an unresolvable ref as-is", func(t *testing.T) {
		t.Parallel()

		vars := ast.NewVars()
		vars.Set("ALLOWED_ENVS", ast.Var{Value: []any{"dev", "staging", "prod"}})

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Ref: ".NONEXISTENT"}}

		require.Empty(t, getEnumValues(e.resolveEnumRefForPrompt(v, vars, dir).Enum))
	})

	t.Run("passes through a static enum unchanged", func(t *testing.T) {
		t.Parallel()

		v := &ast.VarsWithValidation{Name: "ENV", Enum: &ast.Enum{Value: []string{"a", "b"}}}

		require.Same(t, v, e.resolveEnumRefForPrompt(v, ast.NewVars(), dir))
	})

	t.Run("resolves a ref to a dynamic sh variable", func(t *testing.T) {
		t.Parallel()

		// FastGetVariables stores un-evaluated dynamic vars as {Value: "", Sh: ...}.
		// The sh command must still be evaluated so the ref resolves.
		fastVars := ast.NewVars()
		fastVars.Set("AVAILABLE_SERVICES", ast.Var{Value: "", Sh: strPtr("printf 'api\nweb\ndb\n'")})

		v := &ast.VarsWithValidation{Name: "SERVICE", Enum: &ast.Enum{Ref: ".AVAILABLE_SERVICES | splitLines | compact"}}

		resolved := e.resolveEnumRefForPrompt(v, fastVars, dir)

		require.Equal(t, []string{"api", "web", "db"}, getEnumValues(resolved.Enum))
		require.Empty(t, v.Enum.Value, "input var must not be mutated")
	})

	t.Run("keeps free-form fallback when a dynamic ref is empty", func(t *testing.T) {
		t.Parallel()

		fastVars := ast.NewVars()
		fastVars.Set("AVAILABLE_SERVICES", ast.Var{Value: "", Sh: strPtr("printf ''")})

		v := &ast.VarsWithValidation{Name: "SERVICE", Enum: &ast.Enum{Ref: ".AVAILABLE_SERVICES | splitLines | compact"}}

		require.Empty(t, getEnumValues(e.resolveEnumRefForPrompt(v, fastVars, dir).Enum))
	})

	t.Run("resolves a dynamic ref even when an unrelated sh var fails", func(t *testing.T) {
		t.Parallel()

		fastVars := ast.NewVars()
		fastVars.Set("BROKEN", ast.Var{Value: "", Sh: strPtr("exit 1")})
		fastVars.Set("AVAILABLE_SERVICES", ast.Var{Value: "", Sh: strPtr("printf 'api\nweb\ndb\n'")})

		v := &ast.VarsWithValidation{Name: "SERVICE", Enum: &ast.Enum{Ref: ".AVAILABLE_SERVICES | splitLines | compact"}}

		resolved := e.resolveEnumRefForPrompt(v, fastVars, dir)

		require.Equal(t, []string{"api", "web", "db"}, getEnumValues(resolved.Enum))
	})
}

func strPtr(s string) *string {
	return &s
}
