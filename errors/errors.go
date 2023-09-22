package errors

import "errors"

// General exit codes
const (
	CodeOk      int = iota // Used when the program exits without errors
	CodeUnknown            // Used when no other exit code is appropriate
)

// Taskfile related exit codes
const (
	CodeTaskfileNotFound int = iota + 100
	CodeTaskfileAlreadyExists
	CodeTaskfileInvalid
	CodeTaskfileFetchFailed
	CodeTaskfileNotTrusted
	CodeTaskfileNotSecure
	CodeTaskfileCacheNotFound
	CodeTaskfileVersionNotDefined
	CodeTaskfileNetworkTimeout
)

// Task related exit codes
const (
	CodeTaskNotFound int = iota + 200
	CodeTaskRunError
	CodeTaskInternal
	CodeTaskNameConflict
	CodeTaskCalledTooManyTimes
	CodeTaskCancelled
	CodeTaskMissingRequiredVars
)

// TaskError extends the standard error interface with a Code method. This code will
// be used as the exit code of the program which allows the user to distinguish
// between different types of errors.
type TaskError interface {
	error
	Code() int
}

// New returns an error that formats as the given text. Each call to New returns
// a distinct error value even if the text is identical. This wraps the standard
// errors.New function so that we don't need to alias that package.
func New(text string) error {
	return errors.New(text)
}

// Is wraps the standard errors.Is function so that we don't need to alias that package.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As wraps the standard errors.As function so that we don't need to alias that package.
func As(err error, target any) bool {
	return errors.As(err, target)
}
