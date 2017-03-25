package task_test

import (
	"testing"

	"github.com/go-task/task"
)

func TestCyclicDepCheck(t *testing.T) {
	isCyclic := map[string]*task.Task{
		"task-a": &task.Task{
			Deps: []string{"task-b"},
		},
		"task-b": &task.Task{
			Deps: []string{"task-a"},
		},
	}

	if !task.HasCyclicDep(isCyclic) {
		t.Error("Task should be cyclic")
	}

	isNotCyclic := map[string]*task.Task{
		"task-a": &task.Task{
			Deps: []string{"task-c"},
		},
		"task-b": &task.Task{
			Deps: []string{"task-c"},
		},
		"task-c": &task.Task{},
	}

	if task.HasCyclicDep(isNotCyclic) {
		t.Error("Task should not be cyclic")
	}
}
