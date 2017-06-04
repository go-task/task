package task_test

import (
	"testing"

	"github.com/go-task/task"
)

func TestCyclicDepCheck(t *testing.T) {
	isCyclic := &task.Executor{
		Tasks: task.Tasks{
			"task-a": &task.Task{
				Deps: []string{"task-b"},
			},
			"task-b": &task.Task{
				Deps: []string{"task-a"},
			},
		},
	}

	if !isCyclic.HasCyclicDep() {
		t.Error("Task should be cyclic")
	}

	isNotCyclic := &task.Executor{
		Tasks: task.Tasks{
			"task-a": &task.Task{
				Deps: []string{"task-c"},
			},
			"task-b": &task.Task{
				Deps: []string{"task-c"},
			},
			"task-c": &task.Task{},
		},
	}

	if isNotCyclic.HasCyclicDep() {
		t.Error("Task should not be cyclic")
	}
}
