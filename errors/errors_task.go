package errors

import (
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// TaskNotFoundError is returned when the specified task is not found in the
// Taskfile.
type TaskNotFoundError struct {
	TaskName   string
	DidYouMean string
}

func (err *TaskNotFoundError) Error() string {
	if err.DidYouMean != "" {
		return fmt.Sprintf(
			`task: Task %q does not exist. Did you mean %q?`,
			err.TaskName,
			err.DidYouMean,
		)
	}

	return fmt.Sprintf(`task: Task %q does not exist`, err.TaskName)
}

func (err *TaskNotFoundError) Code() int {
	return CodeTaskNotFound
}

// TaskRunError is returned when a command in a task returns a non-zero exit
// code.
type TaskRunError struct {
	TaskName string
	Err      error
}

func (err *TaskRunError) Error() string {
	return fmt.Sprintf(`task: Failed to run task %q: %v`, err.TaskName, err.Err)
}

func (err *TaskRunError) Code() int {
	return CodeTaskRunError
}

func (err *TaskRunError) TaskExitCode() int {
	if c, ok := interp.IsExitStatus(err.Err); ok {
		return int(c)
	}
	return err.Code()
}

// TaskInternalError when the user attempts to invoke a task that is internal.
type TaskInternalError struct {
	TaskName string
}

func (err *TaskInternalError) Error() string {
	return fmt.Sprintf(`task: Task "%s" is internal`, err.TaskName)
}

func (err *TaskInternalError) Code() int {
	return CodeTaskInternal
}

// TaskNameConflictError is returned when multiple tasks with the same name or
// alias are found.
type TaskNameConflictError struct {
	AliasName string
	TaskNames []string
}

func (err *TaskNameConflictError) Error() string {
	return fmt.Sprintf(`task: Multiple tasks (%s) with alias %q found`, strings.Join(err.TaskNames, ", "), err.AliasName)
}

func (err *TaskNameConflictError) Code() int {
	return CodeTaskNameConflict
}

// TaskCalledTooManyTimesError is returned when the maximum task call limit is
// exceeded. This is to prevent infinite loops and cyclic dependencies.
type TaskCalledTooManyTimesError struct {
	TaskName        string
	MaximumTaskCall int
}

func (err *TaskCalledTooManyTimesError) Error() string {
	return fmt.Sprintf(
		`task: Maximum task call exceeded (%d) for task %q: probably an cyclic dep or infinite loop`,
		err.MaximumTaskCall,
		err.TaskName,
	)
}

func (err *TaskCalledTooManyTimesError) Code() int {
	return CodeTaskCalledTooManyTimes
}

// TaskCancelledByUserError is returned when the user does not accept an optional prompt to continue.
type TaskCancelledByUserError struct {
	TaskName string
}

func (err *TaskCancelledByUserError) Error() string {
	return fmt.Sprintf(`task: Task %q cancelled by user`, err.TaskName)
}

func (err *TaskCancelledByUserError) Code() int {
	return CodeTaskCancelled
}

// TaskCancelledNoTerminalError is returned when trying to run a task with a prompt in a non-terminal environment.
type TaskCancelledNoTerminalError struct {
	TaskName string
}

func (err *TaskCancelledNoTerminalError) Error() string {
	return fmt.Sprintf(
		`task: Task %q cancelled because it has a prompt and the environment is not a terminal. Use --yes (-y) to run anyway.`,
		err.TaskName,
	)
}

func (err *TaskCancelledNoTerminalError) Code() int {
	return CodeTaskCancelled
}

// TaskMissingRequiredVars is returned when a task is missing required variables.
type TaskMissingRequiredVars struct {
	TaskName    string
	MissingVars []string
}

func (err *TaskMissingRequiredVars) Error() string {
	return fmt.Sprintf(
		`task: Task %q cancelled because it is missing required variables: %s`,
		err.TaskName,
		strings.Join(err.MissingVars, ", "),
	)
}

func (err *TaskMissingRequiredVars) Code() int {
	return CodeTaskMissingRequiredVars
}
