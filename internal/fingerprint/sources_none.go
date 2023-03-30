package fingerprint

import "github.com/go-task/task/v3/taskfile"

// NoneChecker is a no-op Checker.
// It will always report that the task is not up-to-date.
type NoneChecker struct{}

func (NoneChecker) IsUpToDate(t *taskfile.Task) (bool, error) {
	return false, nil
}

func (NoneChecker) Value(t *taskfile.Task) (any, error) {
	return "", nil
}

func (NoneChecker) OnError(t *taskfile.Task) error {
	return nil
}

func (NoneChecker) Kind() string {
	return "none"
}
