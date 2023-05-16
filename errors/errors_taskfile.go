package errors

import (
	"fmt"
	"net/http"
)

// TaskfileNotFoundError is returned when no appropriate Taskfile is found when
// searching the filesystem.
type TaskfileNotFoundError struct {
	URI  string
	Walk bool
}

func (err TaskfileNotFoundError) Error() string {
	var walkText string
	if err.Walk {
		walkText = " (or any of the parent directories)"
	}
	return fmt.Sprintf(`task: No Taskfile found at %q%s`, err.URI, walkText)
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
	URI string
	Err error
}

func (err TaskfileInvalidError) Error() string {
	return fmt.Sprintf("task: Failed to parse %s:\n%v", err.URI, err.Err)
}

func (err TaskfileInvalidError) Code() int {
	return CodeTaskfileInvalid
}

// TaskfileFetchFailedError is returned when no appropriate Taskfile is found when
// searching the filesystem.
type TaskfileFetchFailedError struct {
	URI            string
	HTTPStatusCode int
}

func (err TaskfileFetchFailedError) Error() string {
	var statusText string
	if err.HTTPStatusCode != 0 {
		statusText = fmt.Sprintf(" with status code %d (%s)", err.HTTPStatusCode, http.StatusText(err.HTTPStatusCode))
	}
	return fmt.Sprintf(`task: Download of %q failed%s`, err.URI, statusText)
}

func (err TaskfileFetchFailedError) Code() int {
	return CodeTaskfileFetchFailed
}

// TaskfileNotTrustedError is returned when the user does not accept the trust
// prompt when downloading a remote Taskfile.
type TaskfileNotTrustedError struct {
	URI string
}

func (err *TaskfileNotTrustedError) Error() string {
	return fmt.Sprintf(
		`task: Taskfile %q not trusted by user`,
		err.URI,
	)
}

func (err *TaskfileNotTrustedError) Code() int {
	return CodeTaskfileNotTrusted
}

// TaskfileNotSecureError is returned when the user attempts to download a
// remote Taskfile over an insecure connection.
type TaskfileNotSecureError struct {
	URI string
}

func (err *TaskfileNotSecureError) Error() string {
	return fmt.Sprintf(
		`task: Taskfile %q cannot be downloaded over an insecure connection. You can override this by using the --insecure flag`,
		err.URI,
	)
}

func (err *TaskfileNotSecureError) Code() int {
	return CodeTaskfileNotSecure
}
