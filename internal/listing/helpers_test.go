package listing_test

import "github.com/go-task/task/v3/taskfile/ast"

func newTask(name, desc string) *ast.Task {
	return &ast.Task{Task: name, Desc: desc}
}

func taskNames(tasks []*ast.Task) []string {
	names := make([]string, len(tasks))
	for i, t := range tasks {
		names[i] = t.Task
	}
	return names
}
