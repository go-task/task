package errors

import (
	"fmt"
)

// TaskfileNotFoundError is returned when no appropriate Taskfile is found when
// searching the filesystem.
type TaskfileNotFoundError struct {
	Dir  string
	Walk bool
}

func (err TaskfileNotFoundError) Error() string {
	var walkText string
	if err.Walk {
		walkText = " (or any of the parent directories)"
	}
	return fmt.Sprintf(`task: No Taskfile found in "%s"%s. Use "task --init" to create a new one`, err.Dir, walkText)
}

func (err TaskfileNotFoundError) Code() int {
	return CodeTaskfileNotFound
}

// TaskfileAlreadyExistsError is returned on creating a Taskfile if one already
// exists.
type TaskfileAlreadyExistsError struct{}

func (err TaskfileAlreadyExistsError) Error() string {
	return "task: A Taskfile already exists"
}

func (err TaskfileAlreadyExistsError) Code() int {
	return CodeTaskfileAlreadyExists
}

// TaskfileInvalidError is returned when the Taskfile contains syntax errors or
// cannot be parsed for some reason.
type TaskfileInvalidError struct {
	FilePath string
	Err      error
}

func (err TaskfileInvalidError) Error() string {
	return fmt.Sprintf("task: Failed to parse %s:\n%v", err.FilePath, err.Err)
}

func (err TaskfileInvalidError) Code() int {
	return CodeTaskfileInvalid
}
