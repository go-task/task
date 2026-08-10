package task_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
)

func TestCompletion(t *testing.T) {
	t.Parallel()

	for _, shell := range []string{"bash", "zsh", "fish", "powershell", "nu", "nushell"} {
		t.Run(shell, func(t *testing.T) {
			t.Parallel()
			script, err := task.Completion(shell)
			require.NoError(t, err)
			require.NotEmpty(t, script)
		})
	}

	for _, shell := range []string{"", "tcsh"} {
		t.Run("unknown/"+shell, func(t *testing.T) {
			t.Parallel()
			_, err := task.Completion(shell)
			require.ErrorContains(t, err, "unknown shell")
		})
	}
}
