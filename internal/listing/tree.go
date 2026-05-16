package listing

import (
	"strings"

	"github.com/go-task/task/v3/taskfile/ast"
)

// TaskGroup represents a set of tasks under a common top-level namespace.
type TaskGroup struct {
	Namespace string
	Tasks     []*ast.Task
}

func (g TaskGroup) LocalName(t *ast.Task) string {
	if g.Namespace == "" {
		return t.Task
	}
	return strings.TrimPrefix(t.Task, g.Namespace+":")
}

// GroupByNamespace partitions tasks into groups by their top-level namespace.
func GroupByNamespace(tasks []*ast.Task) []TaskGroup {
	groupMap := make(map[string]int)
	var groups []TaskGroup
	for _, t := range tasks {
		ns := topLevelNamespace(t.Task)
		idx, exists := groupMap[ns]
		if !exists {
			idx = len(groups)
			groupMap[ns] = idx
			groups = append(groups, TaskGroup{Namespace: ns})
		}
		groups[idx].Tasks = append(groups[idx].Tasks, t)
	}
	return groups
}

func HasNamespacedGroups(groups []TaskGroup) bool {
	for _, g := range groups {
		if g.Namespace != "" {
			return true
		}
	}
	return false
}

func HasRootGroup(groups []TaskGroup) bool {
	for _, g := range groups {
		if g.Namespace == "" {
			return true
		}
	}
	return false
}

func topLevelNamespace(taskName string) string {
	if ns, _, found := strings.Cut(taskName, ":"); found {
		return ns
	}
	return ""
}
