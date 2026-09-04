package task

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskResult(t *testing.T) {
	t.Parallel()

	live := context.Background()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	failure := errors.New("exit status 1")

	tests := []struct {
		name    string
		ctx     context.Context
		skipped bool
		err     error
		want    TaskResult
	}{
		{"no error", live, false, nil, TaskSucceeded},
		{"failed", live, false, failure, TaskFailed},
		{"skipped", live, true, nil, TaskSkipped},
		// A killed process reports the context error on Unix but a plain exit
		// status on Windows, so the context decides, not the error.
		{"killed, reported as a context error", canceled, false, fmt.Errorf("run: %w", context.Canceled), TaskCanceled},
		{"killed, reported as an exit status", canceled, false, failure, TaskCanceled},
		// Succeeding in a context that is already done is still success.
		{"finished before cancellation landed", canceled, false, nil, TaskSucceeded},
		// Skipping wins: Task chose not to run it, so there is nothing to cancel.
		{"skipped in a cancelled context", canceled, true, nil, TaskSkipped},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, taskResult(test.ctx, test.skipped, test.err))
		})
	}
}

func TestTaskResultString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "succeeded", TaskSucceeded.String())
	assert.Equal(t, "failed", TaskFailed.String())
	assert.Equal(t, "canceled", TaskCanceled.String())
	assert.Equal(t, "skipped", TaskSkipped.String())
}
