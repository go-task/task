package errors

import "testing"

func TestTaskRunErrorCodeUnwrapsCancelled(t *testing.T) {
	t.Parallel()
	inner := &TaskCancelledByUserError{TaskName: "prompted"}
	err := &TaskRunError{TaskName: "prompted", Err: inner}
	if got := err.Code(); got != CodeTaskCancelled {
		t.Fatalf("Code() = %d, want %d", got, CodeTaskCancelled)
	}
}

func TestTaskRunErrorCodeStaysRunErrorWithoutInnerTaskError(t *testing.T) {
	t.Parallel()
	err := &TaskRunError{TaskName: "foo", Err: New("command failed")}
	if got := err.Code(); got != CodeTaskRunError {
		t.Fatalf("Code() = %d, want %d", got, CodeTaskRunError)
	}
}
