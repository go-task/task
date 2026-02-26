package errors

import (
	"errors"
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
	var exit interp.ExitStatus
	if errors.As(err.Err, &exit) {
		return int(exit)
	}
	return err.Code()
}

func (err *TaskRunError) Unwrap() error {
	return err.Err
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

// TaskNameConflictError is returned when multiple tasks with a matching name or
// alias are found.
type TaskNameConflictError struct {
	Call      string
	TaskNames []string
}

func (err *TaskNameConflictError) Error() string {
	return fmt.Sprintf(`task: Found multiple tasks (%s) that match %q`, strings.Join(err.TaskNames, ", "), err.Call)
}

func (err *TaskNameConflictError) Code() int {
	return CodeTaskNameConflict
}

type TaskNameFlattenConflictError struct {
	TaskName string
	Include  string
}

func (err *TaskNameFlattenConflictError) Error() string {
	return fmt.Sprintf(`task: Found multiple tasks (%s) included by "%s""`, err.TaskName, err.Include)
}

func (err *TaskNameFlattenConflictError) Code() int {
	return CodeTaskNameConflict
}

// TaskCyclicExecutionDetectedError is returned when the Execution Graph detects
// a cyclic execution condition.
type TaskCyclicExecutionDetectedError struct {
	TaskName        string
	CallingTaskName string
}

func (err *TaskCyclicExecutionDetectedError) Error() string {
	if len(err.CallingTaskName) > 0 {
		return fmt.Sprintf(
			`task: Cyclic task call execution detected for task %q (calling task %q)`,
			err.TaskName,
			err.CallingTaskName,
		)
	} else {
		return fmt.Sprintf(
			`task: Cyclic task call execution detected for task %q`,
			err.TaskName,
		)
	}
}

func (err *TaskCyclicExecutionDetectedError) Code() int {
	return CodeTaskCyclicExecutionDetected
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

// TaskMissingRequiredVarsError is returned when a task is missing required variables.

type MissingVar struct {
	Name          string
	AllowedValues []string
}
type TaskMissingRequiredVarsError struct {
	TaskName    string
	MissingVars []MissingVar
}

func (v MissingVar) String() string {
	if len(v.AllowedValues) == 0 {
		return v.Name
	}
	return fmt.Sprintf("%s (allowed values: %v)", v.Name, v.AllowedValues)
}

func (err *TaskMissingRequiredVarsError) Error() string {
	vars := make([]string, 0, len(err.MissingVars))
	for _, v := range err.MissingVars {
		vars = append(vars, v.String())
	}

	return fmt.Sprintf(
		`task: Task %q cancelled because it is missing required variables: %s`,
		err.TaskName,
		strings.Join(vars, ", "))
}

func (err *TaskMissingRequiredVarsError) Code() int {
	return CodeTaskMissingRequiredVars
}

type NotAllowedVar struct {
	Value string
	Enum  []string
	Name  string
}

type TaskNotAllowedVarsError struct {
	TaskName       string
	NotAllowedVars []NotAllowedVar
}

func (err *TaskNotAllowedVarsError) Error() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("task: Task %q cancelled because it is missing required variables:\n", err.TaskName)) //nolint:staticcheck
	for _, s := range err.NotAllowedVars {
		builder.WriteString(fmt.Sprintf("  - %s has an invalid value : '%s' (allowed values : %v)\n", s.Name, s.Value, s.Enum)) //nolint:staticcheck
	}

	return builder.String()
}

func (err *TaskNotAllowedVarsError) Code() int {
	return CodeTaskNotAllowedVars
}
