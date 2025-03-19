package fingerprint

import (
	"context"

	"github.com/go-task/task/v3/taskfile/ast"
)

// StatusCheckable defines any type that can check if the status of a task is up-to-date.
type StatusCheckable interface {
	IsUpToDate(ctx context.Context, t *ast.Task) (bool, error)
}

// SourcesCheckable defines any type that can check if the sources of a task are up-to-date.
type SourcesCheckable interface {
	SetUpToDate(t *ast.Task, sourceState string) error
	IsUpToDate(t *ast.Task) (upToDate bool, sourceState string, err error)
	Value(t *ast.Task) (any, error)
	OnError(t *ast.Task) error
	Kind() string
}
