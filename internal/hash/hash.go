package hash

import (
	"fmt"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/go-task/task/v3/taskfile"
)

type HashFunc func(*taskfile.Task) (string, error)

func Empty(*taskfile.Task) (string, error) {
	return "", nil
}

func Name(t *taskfile.Task) (string, error) {
	return t.Task, nil
}

func Hash(t *taskfile.Task) (string, error) {
	h, err := hashstructure.Hash(t, hashstructure.FormatV2, nil)
	return fmt.Sprintf("%s:%d", t.Task, h), err
}
