package errors

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"go.yaml.in/yaml/v4"
)

type (
	TaskfileDecodeError struct {
		Message  string
		Location string
		Line     int
		Column   int
		Tag      string
		Snippet  string
		Err      error
	}
)

func NewTaskfileDecodeError(err error, node *yaml.Node) *TaskfileDecodeError {
	// If the error is already a DecodeError, return it
	taskfileInvalidErr := &TaskfileDecodeError{}
	if errors.As(err, &taskfileInvalidErr) {
		return taskfileInvalidErr
	}
	return &TaskfileDecodeError{
		Line:   node.Line,
		Column: node.Column,
		Tag:    node.ShortTag(),
		Err:    err,
	}
}

func (err *TaskfileDecodeError) Error() string {
	buf := &bytes.Buffer{}

	// Print the error message
	if err.Message != "" {
		fmt.Fprintln(buf, color.RedString("err:  %s", err.Message))
	} else {
		// Extract the errors from the TypeError
		te := &yaml.TypeError{}
		if errors.As(err.Err, &te) {
			if len(te.Errors) > 1 {
				fmt.Fprintln(buf, color.RedString("errs:"))
				for _, message := range te.Errors {
					fmt.Fprintln(buf, color.RedString("- %s", message.Err.Error()))
				}
			} else {
				fmt.Fprintln(buf, color.RedString("err:  %s", te.Errors[0].Err.Error()))
			}
		} else {
			// Otherwise print the error message normally
			fmt.Fprintln(buf, color.RedString("err:  %s", err.Err))
		}
	}
	fmt.Fprintln(buf, color.RedString("file: %s:%d:%d", err.Location, err.Line, err.Column))
	fmt.Fprint(buf, err.Snippet)
	return buf.String()
}

func (err *TaskfileDecodeError) Debug() string {
	const indentWidth = 2
	buf := &bytes.Buffer{}
	fmt.Fprintln(buf, "TaskfileDecodeError:")

	// Recursively loop through the error chain and print any details
	var debug func(error, int)
	debug = func(err error, indent int) {
		indentStr := strings.Repeat(" ", indent*indentWidth)

		// Nothing left to unwrap
		if err == nil {
			fmt.Fprintf(buf, "%sEnd of chain\n", indentStr)
			return
		}

		// Taskfile decode error
		decodeErr := &TaskfileDecodeError{}
		if errors.As(err, &decodeErr) {
			fmt.Fprintf(buf, "%s%s (%s:%d:%d)\n",
				indentStr,
				cmp.Or(decodeErr.Message, "<no_message>"),
				decodeErr.Location,
				decodeErr.Line,
				decodeErr.Column,
			)
			debug(errors.Unwrap(err), indent+1)
			return
		}

		fmt.Fprintf(buf, "%s%s\n", indentStr, err)
		debug(errors.Unwrap(err), indent+1)
	}
	debug(err, 0)
	return buf.String()
}

func (err *TaskfileDecodeError) Unwrap() error {
	return err.Err
}

func (err *TaskfileDecodeError) Code() int {
	return CodeTaskfileDecode
}

func (err *TaskfileDecodeError) WithMessage(format string, a ...any) *TaskfileDecodeError {
	err.Message = fmt.Sprintf(format, a...)
	return err
}

func (err *TaskfileDecodeError) WithTypeMessage(t string) *TaskfileDecodeError {
	err.Message = fmt.Sprintf("cannot unmarshal %s into %s", err.Tag, t)
	return err
}

func (err *TaskfileDecodeError) WithFileInfo(location string, snippet string) *TaskfileDecodeError {
	err.Location = location
	err.Snippet = snippet
	return err
}
