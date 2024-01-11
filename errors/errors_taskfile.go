package errors

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Masterminds/semver/v3"
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

// TaskfileCacheNotFoundError is returned when the user attempts to use an offline
// (cached) Taskfile but the files does not exist in the cache.
type TaskfileCacheNotFoundError struct {
	URI string
}

func (err *TaskfileCacheNotFoundError) Error() string {
	return fmt.Sprintf(
		`task: Taskfile %q was not found in the cache. Remove the --offline flag to use a remote copy or download it using the --download flag`,
		err.URI,
	)
}

func (err *TaskfileCacheNotFoundError) Code() int {
	return CodeTaskfileCacheNotFound
}

// TaskfileVersionCheckError is returned when the user attempts to run a
// Taskfile that does not contain a Taskfile schema version key or if they try
// to use a feature that is not supported by the schema version.
type TaskfileVersionCheckError struct {
	URI           string
	SchemaVersion *semver.Version
	Message       string
}

func (err *TaskfileVersionCheckError) Error() string {
	if err.SchemaVersion == nil {
		return fmt.Sprintf(
			`task: Missing schema version in Taskfile %q`,
			err.URI,
		)
	}
	return fmt.Sprintf(
		"task: Invalid schema version in Taskfile %q:\nSchema version (%s) %s",
		err.URI,
		err.SchemaVersion.String(),
		err.Message,
	)
}

func (err *TaskfileVersionCheckError) Code() int {
	return CodeTaskfileVersionCheckError
}

// TaskfileNetworkTimeoutError is returned when the user attempts to use a remote
// Taskfile but a network connection could not be established within the timeout.
type TaskfileNetworkTimeoutError struct {
	URI          string
	Timeout      time.Duration
	CheckedCache bool
}

func (err *TaskfileNetworkTimeoutError) Error() string {
	var cacheText string
	if err.CheckedCache {
		cacheText = " and no offline copy was found in the cache"
	}
	return fmt.Sprintf(
		`task: Network connection timed out after %s while attempting to download Taskfile %q%s`,
		err.Timeout, err.URI, cacheText,
	)
}

func (err *TaskfileNetworkTimeoutError) Code() int {
	return CodeTaskfileNetworkTimeout
}
