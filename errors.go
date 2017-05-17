package task

import (
	"fmt"
)

type taskFileNotFound struct {
	taskFile string
}

func (err taskFileNotFound) Error() string {
	return fmt.Sprintf(`task: No task file found (is it named "%s"?). Use "task --init" to create a new one`, err.taskFile)
}

type taskNotFoundError struct {
	taskName string
}

func (err *taskNotFoundError) Error() string {
	return fmt.Sprintf(`task: Task "%s" not found`, err.taskName)
}

type taskRunError struct {
	taskName string
	err      error
}

func (err *taskRunError) Error() string {
	return fmt.Sprintf(`task: Failed to run task "%s": %v`, err.taskName, err.err)
}

type cyclicDepError struct {
	taskName string
}

func (err *cyclicDepError) Error() string {
	return fmt.Sprintf(`task: Cyclic dependency of task "%s" detected`, err.taskName)
}

type cantWatchNoSourcesError struct {
	taskName string
}

func (err *cantWatchNoSourcesError) Error() string {
	return fmt.Sprintf(`task: Can't watch task "%s" because it has no specified sources`, err.taskName)
}
