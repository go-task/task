package task

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinishError(t *testing.T) {
	t.Parallel()

	liveCtx := context.Background()
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	timeout := errors.New("task timed out")
	timedOutCtx, cancelTimedOut := context.WithCancelCause(context.Background())
	cancelTimedOut(timeout)
	t.Cleanup(func() { cancelTimedOut(nil) })

	// A killed process reports the context error on Unix but a plain exit status
	// on Windows, so both shapes must classify as cancellation.
	unixShape := fmt.Errorf("running command: %w", context.Canceled)
	windowsShape := errors.New("exit status 1")

	t.Run("succeeded", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, finishError(canceledCtx, nil))
	})

	t.Run("failed while the context was live", func(t *testing.T) {
		t.Parallel()
		err := errors.New("exit status 1")
		assert.Equal(t, err, finishError(liveCtx, err))
	})

	t.Run("canceled reported by the interpreter", func(t *testing.T) {
		t.Parallel()
		got := finishError(canceledCtx, unixShape)
		require.ErrorIs(t, got, context.Canceled)
		// Already carries the cause, so it is passed through unwrapped.
		assert.Equal(t, unixShape, got)
	})

	t.Run("canceled reported as a plain exit status", func(t *testing.T) {
		t.Parallel()
		got := finishError(canceledCtx, windowsShape)
		require.ErrorIs(t, got, context.Canceled)
		assert.ErrorIs(t, got, windowsShape)
		assert.Contains(t, got.Error(), "exit status 1")
	})

	t.Run("cancellation cause wins over the generic error", func(t *testing.T) {
		t.Parallel()
		got := finishError(timedOutCtx, windowsShape)
		require.ErrorIs(t, got, timeout)
		assert.ErrorIs(t, got, windowsShape)
	})
}
