package hash

import (
	"fmt"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/go-task/task/v3/taskfile/ast"
)

type HashFunc func(*ast.Task) (string, error)

func Empty(*ast.Task) (string, error) {
	return "", nil
}

func Name(t *ast.Task) (string, error) {
	return t.Task, nil
}

func Hash(t *ast.Task) (string, error) {
	h, err := hashstructure.Hash(t, hashstructure.FormatV2, nil)
	return fmt.Sprintf("%s:%d", t.Task, h), err
}
