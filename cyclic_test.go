package task_test

import (
	"testing"

	"github.com/go-task/task"

	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, task.ErrCyclicDepDetected, isCyclic.CheckCyclicDep(), "task should be cyclic")

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

	assert.NoError(t, isNotCyclic.CheckCyclicDep())

	inexixtentTask := &task.Executor{
		Tasks: task.Tasks{
			"task-a": &task.Task{
				Deps: []*task.Dep{&task.Dep{Task: "invalid-task"}},
			},
		},
	}

	// FIXME: by now Task should ignore non existent tasks
	// in the future we should improve the detection of
	// tasks called with interpolation?
	//     task:
	//       deps:
	//         - task: "task{{.VARIABLE}}"
	//       vars:
	//         VARIABLE: something
	assert.NoError(t, inexixtentTask.CheckCyclicDep())
}
