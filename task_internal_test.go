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
		want    Result
	}{
		{"no error", live, false, nil, ResultSucceeded},
		{"failed", live, false, failure, ResultFailed},
		{"skipped", live, true, nil, ResultSkipped},
		// A killed process reports the context error on Unix but a plain exit
		// status on Windows, so the context decides, not the error.
		{"killed, reported as a context error", canceled, false, fmt.Errorf("run: %w", context.Canceled), ResultCanceled},
		{"killed, reported as an exit status", canceled, false, failure, ResultCanceled},
		// Succeeding in a context that is already done is still success.
		{"finished before cancellation landed", canceled, false, nil, ResultSucceeded},
		// Skipping wins: Task chose not to run it, so there is nothing to cancel.
		{"skipped in a cancelled context", canceled, true, nil, ResultSkipped},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, taskResult(test.ctx, test.skipped, test.err))
		})
	}
}

func TestResultString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "succeeded", ResultSucceeded.String())
	assert.Equal(t, "failed", ResultFailed.String())
	assert.Equal(t, "canceled", ResultCanceled.String())
	assert.Equal(t, "skipped", ResultSkipped.String())
}
