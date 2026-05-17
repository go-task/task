package listing_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/listing"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestFormatRequires_Simple(t *testing.T) {
	t.Parallel()
	req := &ast.Requires{Vars: []*ast.VarsWithValidation{{Name: "VERSION"}, {Name: "TAG"}}}
	require.Equal(t, "VERSION, TAG", listing.FormatRequires(req))
}

func TestFormatRequires_WithEnum(t *testing.T) {
	t.Parallel()
	req := &ast.Requires{Vars: []*ast.VarsWithValidation{
		{Name: "REGISTRY", Enum: &ast.Enum{Value: []string{"ecr", "gcr", "dockerhub"}}}, {Name: "TAG"},
	}}
	require.Equal(t, "REGISTRY (enum: ecr, gcr, dockerhub), TAG", listing.FormatRequires(req))
}

func TestFormatRequires_Nil(t *testing.T) {
	t.Parallel()
	require.Equal(t, "", listing.FormatRequires(nil))
}

func TestFormatRequires_Empty(t *testing.T) {
	t.Parallel()
	require.Equal(t, "", listing.FormatRequires(&ast.Requires{}))
}

func TestFormatDeps(t *testing.T) {
	t.Parallel()
	deps := []*ast.Dep{{Task: "lint"}, {Task: "test:unit"}}
	require.Equal(t, "lint, test:unit", listing.FormatDeps(deps))
}

func TestFormatDeps_Empty(t *testing.T) {
	t.Parallel()
	require.Equal(t, "", listing.FormatDeps(nil))
}

func TestHasRequires_True(t *testing.T) {
	t.Parallel()
	task := &ast.Task{Task: "build", Requires: &ast.Requires{Vars: []*ast.VarsWithValidation{{Name: "V"}}}}
	require.True(t, listing.HasRequires(task))
}

func TestHasRequires_False(t *testing.T) {
	t.Parallel()
	require.False(t, listing.HasRequires(&ast.Task{Task: "build"}))
}
