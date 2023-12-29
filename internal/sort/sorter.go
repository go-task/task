package sort

import (
	"sort"
	"strings"

	"github.com/go-task/task/v3/taskfile/ast"
)

type TaskSorter interface {
	Sort([]*ast.Task)
}

type Noop struct{}

func (s *Noop) Sort(tasks []*ast.Task) {}

type AlphaNumeric struct{}

// Tasks that are not namespaced should be listed before tasks that are.
// We detect this by searching for a ':' in the task name.
func (s *AlphaNumeric) Sort(tasks []*ast.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Task < tasks[j].Task
	})
}

type AlphaNumericWithRootTasksFirst struct{}

// Tasks that are not namespaced should be listed before tasks that are.
// We detect this by searching for a ':' in the task name.
func (s *AlphaNumericWithRootTasksFirst) Sort(tasks []*ast.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		iContainsColon := strings.Contains(tasks[i].Task, ":")
		jContainsColon := strings.Contains(tasks[j].Task, ":")
		if iContainsColon == jContainsColon {
			return tasks[i].Task < tasks[j].Task
		}
		if !iContainsColon && jContainsColon {
			return true
		}
		return false
	})
}
