package listing

import (
	"path"
	"strings"

	"github.com/go-task/task/v3/taskfile/ast"
)

func IsGlobPattern(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// FilterTasks returns tasks whose name or description matches the pattern.
func FilterTasks(tasks []*ast.Task, pattern string) []*ast.Task {
	if pattern == "" {
		return tasks
	}
	if IsGlobPattern(pattern) {
		return filterByGlob(tasks, pattern)
	}
	return filterBySubstring(tasks, pattern)
}

func filterBySubstring(tasks []*ast.Task, pattern string) []*ast.Task {
	lower := strings.ToLower(pattern)
	var result []*ast.Task
	for _, t := range tasks {
		nameLower := strings.ToLower(t.Task)
		descLower := strings.ToLower(t.Desc)
		if strings.Contains(nameLower, lower) ||
			strings.Contains(descLower, lower) {
			result = append(result, t)
		}
	}
	return result
}

func filterByGlob(tasks []*ast.Task, pattern string) []*ast.Task {
	lowerPattern := strings.ToLower(pattern)
	var result []*ast.Task
	for _, t := range tasks {
		matched, err := path.Match(lowerPattern, strings.ToLower(t.Task))
		if err != nil {
			return filterBySubstring(tasks, pattern)
		}
		if matched {
			result = append(result, t)
		}
	}
	return result
}
