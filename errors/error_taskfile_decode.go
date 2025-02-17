package errors

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

var typeErrorRegex = regexp.MustCompile(`line \d+: (.*)`)

type (
	TaskfileDecodeError struct {
		Message  string
		Location string
		Line     int
		Column   int
		Tag      string
		Snippet  *Snippet
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
					fmt.Fprintln(buf, color.RedString("- %s", extractTypeErrorMessage(message)))
				}
			} else {
				fmt.Fprintln(buf, color.RedString("err:  %s", extractTypeErrorMessage(te.Errors[0])))
			}
		} else {
			// Otherwise print the error message normally
			fmt.Fprintln(buf, color.RedString("err:  %s", err.Err))
		}
	}
	fmt.Fprintln(buf, color.RedString("file: %s:%d:%d", err.Location, err.Line, err.Column))
	fmt.Fprint(buf, err.Snippet.String())
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

func (err *TaskfileDecodeError) WithFileInfo(location string, snippet *Snippet) *TaskfileDecodeError {
	err.Location = location
	err.Snippet = snippet
	return err
}

func extractTypeErrorMessage(message string) string {
	matches := typeErrorRegex.FindStringSubmatch(message)
	if len(matches) == 2 {
		return matches[1]
	}
	return message
}
