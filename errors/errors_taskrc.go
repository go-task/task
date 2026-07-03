package errors

import "fmt"

type TaskRCNotFoundError struct {
	URI  string
	Walk bool
}

func (err TaskRCNotFoundError) Error() string {
	var walkText string
	if err.Walk {
		walkText = " (or any of the parent directories)"
	}
	return fmt.Sprintf(`task: No Task config file found at %q%s`, err.URI, walkText)
}

func (err TaskRCNotFoundError) Code() int {
	return CodeTaskRCNotFoundError
}
