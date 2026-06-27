package task

import (
	"fmt"
	"sync"
	"testing"

	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

// TestResolveMatrixRefsDoesNotMutateSharedMatrix is a regression test for
// #2890. The *ast.Matrix passed to resolveMatrixRefs is part of the shared,
// cached Task AST, and concurrent invocations of the same task (e.g. via
// parallel deps) all call resolveMatrixRefs with the very same *ast.Matrix.
// If resolveMatrixRefs mutates that matrix in place, concurrent callers race
// on the mutation and can observe a value resolved for a different caller
// (cross-contamination), or trip the race detector.
//
// Run with `go test -race` to catch the race deterministically; it is not
// guaranteed to corrupt output on every unsynchronized run.
func TestResolveMatrixRefsDoesNotMutateSharedMatrix(t *testing.T) {
	t.Parallel()

	sharedMatrix := ast.NewMatrix(
		&ast.MatrixElement{Key: "ARCH", Value: &ast.MatrixRow{Ref: ".ARCH_VAR"}},
	)

	const n = 200
	var wg sync.WaitGroup
	results := make([]string, n)
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			want := "amd64"
			if i%2 != 0 {
				want = "arm64"
			}

			vars := ast.NewVars()
			vars.Set("ARCH_VAR", ast.Var{Value: []any{want}})
			cache := &templater.Cache{Vars: vars}

			resolved, err := resolveMatrixRefs(sharedMatrix, cache)
			if err != nil {
				errs[i] = err
				return
			}
			row, ok := resolved.Get("ARCH")
			if !ok {
				errs[i] = fmt.Errorf("call %d: ARCH row missing from resolved matrix", i)
				return
			}
			if len(row.Value) != 1 {
				errs[i] = fmt.Errorf("call %d: ARCH.Value = %v, want a single-element slice", i, row.Value)
				return
			}
			got, _ := row.Value[0].(string)
			results[i] = got
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		want := "amd64"
		if i%2 != 0 {
			want = "arm64"
		}
		if results[i] != want {
			t.Errorf("call %d: got ARCH=%q, want %q (cross-contamination between concurrent callers)", i, results[i], want)
		}
	}

	// The original, shared matrix must remain unresolved: resolveMatrixRefs
	// must operate on a copy, not the input.
	origRow, ok := sharedMatrix.Get("ARCH")
	if !ok {
		t.Fatal("ARCH row missing from original matrix")
	}
	if origRow.Value != nil {
		t.Errorf("shared input matrix was mutated: ARCH.Value = %v, want nil (Ref rows must be resolved into a copy)", origRow.Value)
	}
	if origRow.Ref != ".ARCH_VAR" {
		t.Errorf("shared input matrix Ref was altered: got %q, want %q", origRow.Ref, ".ARCH_VAR")
	}
}
