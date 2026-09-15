package task_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestPrefixMatching(t *testing.T) {
	tasks := ast.NewTasks(
		&ast.TaskElement{
			Key: "api:openapi:export",
			Value: &ast.Task{
				Task: "api:openapi:export",
			},
		},
		&ast.TaskElement{
			Key: "api:openapi:import",
			Value: &ast.Task{
				Task: "api:openapi:import",
			},
		},
		&ast.TaskElement{
			Key: "docker:build:production",
			Value: &ast.Task{
				Task:    "docker:build:production",
				Aliases: []string{"d:b:prod"},
			},
		},
		&ast.TaskElement{
			Key: "docker:build:staging",
			Value: &ast.Task{
				Task: "docker:build:staging",
			},
		},
		&ast.TaskElement{
			Key: "build",
			Value: &ast.Task{
				Task: "build",
			},
		},
		&ast.TaskElement{
			Key: "internal:secret",
			Value: &ast.Task{
				Task:     "internal:secret",
				Internal: true,
			},
		},
		&ast.TaskElement{
			Key: "wild-*",
			Value: &ast.Task{
				Task: "wild-*",
			},
		},
	)

	taskfile := &ast.Taskfile{
		Tasks: tasks,
	}

	e := &task.Executor{
		Taskfile: taskfile,
	}

	t.Run("Experiment Disabled", func(t *testing.T) {
		// When experiment is not enabled, prefix matching shouldn't happen
		matching, err := e.FindMatchingTasks(&task.Call{Task: "a:o:e"})
		require.NoError(t, err)
		assert.Empty(t, matching)

		matching, err = e.FindMatchingTasks(&task.Call{Task: "b"})
		require.NoError(t, err)
		// "b" should only match if exact task name exists (it doesn't, exact is "build")
		assert.Empty(t, matching)
	})

	t.Run("Experiment Enabled", func(t *testing.T) {
		enableExperimentForTest(t, &experiments.PrefixMatching, 1)

		t.Run("Unique match with equal segments (m=n)", func(t *testing.T) {
			matching, err := e.FindMatchingTasks(&task.Call{Task: "a:o:e"})
			require.NoError(t, err)
			require.Len(t, matching, 1)
			assert.Equal(t, "api:openapi:export", matching[0].Task.Task)
		})

		t.Run("Unique match with fewer segments (m<n)", func(t *testing.T) {
			matching, err := e.FindMatchingTasks(&task.Call{Task: "d:b:s"})
			require.NoError(t, err)
			require.Len(t, matching, 1)
			assert.Equal(t, "docker:build:staging", matching[0].Task.Task)
		})

		t.Run("Unique match single segment", func(t *testing.T) {
			matching, err := e.FindMatchingTasks(&task.Call{Task: "b"})
			require.NoError(t, err)
			require.Len(t, matching, 1)
			assert.Equal(t, "build", matching[0].Task.Task)
		})

		t.Run("Unique match via alias prefix", func(t *testing.T) {
			matching, err := e.FindMatchingTasks(&task.Call{Task: "d:b:p"})
			require.NoError(t, err)
			require.Len(t, matching, 1)
			assert.Equal(t, "docker:build:production", matching[0].Task.Task)
		})

		t.Run("Ambiguous match returns TaskNameConflictError", func(t *testing.T) {
			matching, err := e.FindMatchingTasks(&task.Call{Task: "a:o"})
			assert.Nil(t, matching)
			require.Error(t, err)

			var conflictErr *errors.TaskNameConflictError
			require.ErrorAs(t, err, &conflictErr)
			assert.Equal(t, "a:o", conflictErr.Call)
			assert.ElementsMatch(t, []string{"api:openapi:export", "api:openapi:import"}, conflictErr.TaskNames)
		})

		t.Run("Ambiguous match single segment returns TaskNameConflictError", func(t *testing.T) {
			matching, err := e.FindMatchingTasks(&task.Call{Task: "doc"})
			assert.Nil(t, matching)
			require.Error(t, err)

			var conflictErr *errors.TaskNameConflictError
			require.ErrorAs(t, err, &conflictErr)
			assert.Equal(t, "doc", conflictErr.Call)
			assert.ElementsMatch(t, []string{"docker:build:production", "docker:build:staging"}, conflictErr.TaskNames)
		})

		t.Run("Internal tasks are ignored", func(t *testing.T) {
			matching, err := e.FindMatchingTasks(&task.Call{Task: "int:s"})
			require.NoError(t, err)
			assert.Empty(t, matching)
		})

		t.Run("Exact match takes precedence", func(t *testing.T) {
			matching, err := e.FindMatchingTasks(&task.Call{Task: "build"})
			require.NoError(t, err)
			require.Len(t, matching, 1)
			assert.Equal(t, "build", matching[0].Task.Task)
		})

		t.Run("Wildcard match takes precedence", func(t *testing.T) {
			matching, err := e.FindMatchingTasks(&task.Call{Task: "wild-test"})
			require.NoError(t, err)
			require.Len(t, matching, 1)
			assert.Equal(t, "wild-*", matching[0].Task.Task)
			assert.Equal(t, []string{"test"}, matching[0].Wildcards)
		})

		t.Run("No match returns empty slice without error", func(t *testing.T) {
			matching, err := e.FindMatchingTasks(&task.Call{Task: "nonexistent"})
			require.NoError(t, err)
			assert.Empty(t, matching)
		})
	})
}
