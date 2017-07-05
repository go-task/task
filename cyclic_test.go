package task_test

import (
	"testing"

	"github.com/go-task/task"
)

func TestCyclicDepCheck(t *testing.T) {
	isCyclic := &task.Executor{
		Tasks: task.Tasks{
			"task-a": &task.Task{
				Deps: []*task.Dep{&task.Dep{Task: "task-b"}},
			},
			"task-b": &task.Task{
				Deps: []*task.Dep{&task.Dep{Task: "task-a"}},
			},
		},
	}

	if !isCyclic.HasCyclicDep() {
		t.Error("Task should be cyclic")
	}

	isNotCyclic := &task.Executor{
		Tasks: task.Tasks{
			"task-a": &task.Task{
				Deps: []*task.Dep{&task.Dep{Task: "task-c"}},
			},
			"task-b": &task.Task{
				Deps: []*task.Dep{&task.Dep{Task: "task-c"}},
			},
			"task-c": &task.Task{},
		},
	}

	if isNotCyclic.HasCyclicDep() {
		t.Error("Task should not be cyclic")
	}
}
