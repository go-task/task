package sort

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/taskfile/ast"
)

func TestAlphaNumericWithRootTasksFirst_Sort(t *testing.T) {
	task1 := &ast.Task{Task: "task1"}
	task2 := &ast.Task{Task: "task2"}
	task3 := &ast.Task{Task: "ns1:task3"}
	task4 := &ast.Task{Task: "ns2:task4"}
	task5 := &ast.Task{Task: "task5"}
	task6 := &ast.Task{Task: "ns3:task6"}

	tests := []struct {
		name  string
		tasks []*ast.Task
		want  []*ast.Task
	}{
		{
			name:  "no namespace tasks sorted alphabetically first",
			tasks: []*ast.Task{task3, task2, task1},
			want:  []*ast.Task{task1, task2, task3},
		},
		{
			name:  "namespace tasks sorted alphabetically after non-namespaced tasks",
			tasks: []*ast.Task{task3, task4, task5},
			want:  []*ast.Task{task5, task3, task4},
		},
		{
			name:  "all tasks sorted alphabetically with root tasks first",
			tasks: []*ast.Task{task6, task5, task4, task3, task2, task1},
			want:  []*ast.Task{task1, task2, task5, task3, task4, task6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AlphaNumericWithRootTasksFirst{}
			s.Sort(tt.tasks)
			assert.Equal(t, tt.want, tt.tasks)
		})
	}
}

func TestAlphaNumeric_Sort(t *testing.T) {
	task1 := &ast.Task{Task: "task1"}
	task2 := &ast.Task{Task: "task2"}
	task3 := &ast.Task{Task: "ns1:task3"}
	task4 := &ast.Task{Task: "ns2:task4"}
	task5 := &ast.Task{Task: "task5"}
	task6 := &ast.Task{Task: "ns3:task6"}

	tests := []struct {
		name  string
		tasks []*ast.Task
		want  []*ast.Task
	}{
		{
			name:  "all tasks sorted alphabetically",
			tasks: []*ast.Task{task3, task2, task5, task1, task4, task6},
			want:  []*ast.Task{task3, task4, task6, task1, task2, task5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AlphaNumeric{}
			s.Sort(tt.tasks)
			assert.Equal(t, tt.tasks, tt.want)
		})
	}
}
