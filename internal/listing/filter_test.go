package listing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/internal/listing"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestIsGlobPattern(t *testing.T) {
	t.Parallel()
	assert.True(t, listing.IsGlobPattern("docker:*"))
	assert.True(t, listing.IsGlobPattern("test?"))
	assert.True(t, listing.IsGlobPattern("[ab]"))
	assert.False(t, listing.IsGlobPattern("docker"))
	assert.False(t, listing.IsGlobPattern(""))
}

func TestFilterTasks_EmptyPattern(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{newTask("build", "Build it")}
	result := listing.FilterTasks(tasks, "")
	assert.Equal(t, tasks, result)
}

func TestFilterTasks_Substring(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("docker:build", "Build image"),
		newTask("docker:push", "Push image"),
		newTask("test:unit", "Run unit tests"),
		newTask("lint", "Run linters"),
	}
	result := listing.FilterTasks(tasks, "docker")
	assert.Equal(t, []string{"docker:build", "docker:push"}, taskNames(result))
}

func TestFilterTasks_SubstringCaseInsensitive(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("Docker:Build", "Build image"),
		newTask("lint", "Run linters"),
	}
	result := listing.FilterTasks(tasks, "docker")
	assert.Equal(t, []string{"Docker:Build"}, taskNames(result))
}

func TestFilterTasks_MatchesDescription(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("build", "Build the Docker image"),
		newTask("lint", "Run linters"),
	}
	result := listing.FilterTasks(tasks, "docker")
	assert.Equal(t, []string{"build"}, taskNames(result))
}

func TestFilterTasks_NamespacePrefix(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("docker:build", "Build image"),
		newTask("docker:push", "Push image"),
		newTask("undocker", "Not a namespace match but substring"),
	}
	result := listing.FilterTasks(tasks, "docker")
	assert.Equal(t, []string{"docker:build", "docker:push", "undocker"}, taskNames(result))
}

func TestFilterTasks_Glob(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("docker:build", "Build image"),
		newTask("docker:push", "Push image"),
		newTask("test:unit", "Run unit tests"),
		newTask("lint", "Run linters"),
	}
	result := listing.FilterTasks(tasks, "docker:*")
	assert.Equal(t, []string{"docker:build", "docker:push"}, taskNames(result))
}

func TestFilterTasks_GlobNoMatch(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("build", "Build it"),
		newTask("lint", "Run linters"),
	}
	result := listing.FilterTasks(tasks, "xyz:*")
	assert.Empty(t, result)
}

func TestFilterTasks_NoMatch(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("build", "Build it"),
	}
	result := listing.FilterTasks(tasks, "zzzzz")
	assert.Empty(t, result)
}
