package fingerprint

import "github.com/go-task/task/v3/taskfile/ast"

// NoneChecker is a no-op Checker.
// It will always report that the task is not up-to-date.
type NoneChecker struct{}

func (NoneChecker) IsUpToDate(t *ast.Task) (bool, error) {
	return false, nil
}

func (NoneChecker) Value(t *ast.Task) (any, error) {
	return "", nil
}

func (NoneChecker) OnError(t *ast.Task) error {
	return nil
}

func (NoneChecker) Kind() string {
	return "none"
}
