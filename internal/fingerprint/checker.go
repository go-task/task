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
	IsUpToDate(t *ast.Task) (bool, error)
	Value(t *ast.Task) (any, error)
	OnError(t *ast.Task) error
	Kind() string
}

// DefinitionCheckable defines any type that checks the definition of a task.
type DefinitionCheckable interface {
	HashDefinition(t *ast.Task) (*string, error)
	IsUpToDate(maybeDefinitionPath *string) (bool, error)
	Cleanup(definitionPath *string) error
}
