package listing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/internal/listing"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestGroupByNamespace_Empty(t *testing.T) {
	t.Parallel()
	groups := listing.GroupByNamespace(nil)
	assert.Empty(t, groups)
}

func TestGroupByNamespace_OnlyRoot(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("build", "Build"),
		newTask("test", "Test"),
	}
	groups := listing.GroupByNamespace(tasks)
	assert.Len(t, groups, 1)
	assert.Equal(t, "", groups[0].Namespace)
	assert.Len(t, groups[0].Tasks, 2)
}

func TestGroupByNamespace_OnlyNamespaced(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("ns1:build", "Build"),
		newTask("ns1:test", "Test"),
		newTask("ns2:deploy", "Deploy"),
	}
	groups := listing.GroupByNamespace(tasks)
	assert.Len(t, groups, 2)
	assert.Equal(t, "ns1", groups[0].Namespace)
	assert.Len(t, groups[0].Tasks, 2)
	assert.Equal(t, "ns2", groups[1].Namespace)
	assert.Len(t, groups[1].Tasks, 1)
}

func TestGroupByNamespace_Mixed(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("build", "Build"),
		newTask("ns1:lint", "Lint"),
		newTask("ns1:test", "Test"),
		newTask("deploy", "Deploy"),
	}
	groups := listing.GroupByNamespace(tasks)
	assert.Len(t, groups, 2)
	assert.Equal(t, "", groups[0].Namespace)
	assert.Len(t, groups[0].Tasks, 2)
	assert.Equal(t, "ns1", groups[1].Namespace)
	assert.Len(t, groups[1].Tasks, 2)
}

func TestGroupByNamespace_PreservesOrder(t *testing.T) {
	t.Parallel()
	tasks := []*ast.Task{
		newTask("z:first", ""),
		newTask("a:second", ""),
		newTask("m:third", ""),
	}
	groups := listing.GroupByNamespace(tasks)
	assert.Len(t, groups, 3)
	assert.Equal(t, "z", groups[0].Namespace)
	assert.Equal(t, "a", groups[1].Namespace)
	assert.Equal(t, "m", groups[2].Namespace)
}

func TestLocalName_Namespaced(t *testing.T) {
	t.Parallel()
	g := listing.TaskGroup{Namespace: "ns1"}
	task := newTask("ns1:build", "Build")
	assert.Equal(t, "build", g.LocalName(task))
}

func TestLocalName_Root(t *testing.T) {
	t.Parallel()
	g := listing.TaskGroup{Namespace: ""}
	task := newTask("build", "Build")
	assert.Equal(t, "build", g.LocalName(task))
}

func TestHasNamespacedGroups(t *testing.T) {
	t.Parallel()
	assert.False(t, listing.HasNamespacedGroups([]listing.TaskGroup{{Namespace: ""}}))
	assert.True(t, listing.HasNamespacedGroups([]listing.TaskGroup{{Namespace: "ns"}}))
	assert.True(t, listing.HasNamespacedGroups([]listing.TaskGroup{{Namespace: ""}, {Namespace: "ns"}}))
}

func TestHasRootGroup(t *testing.T) {
	t.Parallel()
	assert.True(t, listing.HasRootGroup([]listing.TaskGroup{{Namespace: ""}}))
	assert.False(t, listing.HasRootGroup([]listing.TaskGroup{{Namespace: "ns"}}))
	assert.True(t, listing.HasRootGroup([]listing.TaskGroup{{Namespace: ""}, {Namespace: "ns"}}))
}
