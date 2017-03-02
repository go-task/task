package task

import (
	"fmt"
)

// ErrNoTaskFile is returned when the program can not find a proper TaskFile
var ErrNoTaskFile = fmt.Errorf(`No task file found (is it named "%s"?)`, TaskFilePath)

type taskNotFoundError struct {
	taskName string
}

func (err *taskNotFoundError) Error() string {
	return fmt.Sprintf(`Task "%s" not found`, err.taskName)
}

type taskRunError struct {
	taskName string
	err      error
}

func (err *taskRunError) Error() string {
	return fmt.Sprintf(`Failed to run task "%s": %v`, err.taskName, err.err)
}

type cyclicDepError struct {
	taskName string
}

func (err *cyclicDepError) Error() string {
	return fmt.Sprintf(`Cyclic dependency of task "%s" detected`, err.taskName)
}
