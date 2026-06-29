package task

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

// TestResolveMatrixRefsDoesNotMutateInput is a regression test for #2890. The
// *ast.Matrix passed to resolveMatrixRefs is part of the shared, cached Task
// AST: the same *ast.Matrix is reused on every invocation of a task. If
// resolveMatrixRefs resolved `ref:` rows in place, concurrent invocations of
// the same task (e.g. via parallel deps) would race on that mutation and leak
// a value resolved for one caller into another caller's expansion.
//
// The invariant that prevents this is that resolveMatrixRefs must resolve into
// a copy and leave its input untouched, which this test asserts deterministically.
func TestResolveMatrixRefsDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	matrix := ast.NewMatrix(
		&ast.MatrixElement{Key: "ARCH", Value: &ast.MatrixRow{Ref: ".ARCH_VAR"}},
	)

	vars := ast.NewVars()
	vars.Set("ARCH_VAR", ast.Var{Value: []any{"amd64"}})
	cache := &templater.Cache{Vars: vars}

	resolved, err := resolveMatrixRefs(matrix, cache)
	require.NoError(t, err)

	// The returned matrix has the ref resolved...
	row, ok := resolved.Get("ARCH")
	require.True(t, ok, "ARCH row missing from resolved matrix")
	require.Equal(t, []any{"amd64"}, row.Value)

	// ...but the shared input matrix must be left untouched.
	orig, ok := matrix.Get("ARCH")
	require.True(t, ok, "ARCH row missing from input matrix")
	require.Nil(t, orig.Value, "input matrix was mutated: Ref rows must be resolved into a copy")
	require.Equal(t, ".ARCH_VAR", orig.Ref, "input matrix Ref was altered")
}
