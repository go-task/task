package errors

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

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
		Snippet  TaskfileSnippet
		Err      error
	}
	TaskfileSnippet struct {
		Lines     []string
		StartLine int
		EndLine   int
		Padding   int
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

	// Print the snippet
	maxLineNumberDigits := digits(err.Snippet.EndLine)
	lineNumberSpacer := strings.Repeat(" ", maxLineNumberDigits)
	columnSpacer := strings.Repeat(" ", err.Column-1)
	for i, line := range err.Snippet.Lines {
		currentLine := err.Snippet.StartLine + i + 1

		lineIndicator := " "
		if currentLine == err.Line {
			lineIndicator = ">"
		}
		columnIndicator := "^"

		// Colors
		line = highlight(line)
		lineIndicator = color.RedString(lineIndicator)
		columnIndicator = color.RedString(columnIndicator)
		lineNumberFormat := fmt.Sprintf("%%%dd", maxLineNumberDigits)
		lineNumber := fmt.Sprintf(lineNumberFormat, currentLine)
		fmt.Fprintf(buf, "%s %s | %s", lineIndicator, lineNumber, line)

		// Print the column indicator
		if currentLine == err.Line {
			fmt.Fprintf(buf, "\n  %s | %s%s", lineNumberSpacer, columnSpacer, columnIndicator)
		}

		// If there are more lines to print, add a newline
		if i < len(err.Snippet.Lines)-1 {
			fmt.Fprintln(buf)
		}
	}

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

func (err *TaskfileDecodeError) WithFileInfo(location string, b []byte, padding int) *TaskfileDecodeError {
	lines := strings.Split(string(b), "\n")
	start := max(err.Line-1-padding, 0)
	end := min(err.Line+padding, len(lines)-1)

	err.Location = location
	err.Snippet = TaskfileSnippet{
		Lines:     lines[start:end],
		StartLine: start,
		EndLine:   end,
		Padding:   padding,
	}

	return err
}

func extractTypeErrorMessage(message string) string {
	matches := typeErrorRegex.FindStringSubmatch(message)
	if len(matches) == 2 {
		return matches[1]
	}
	return message
}

func highlight(line string) string {
	keyRE := regexp.MustCompile(`(.*-? +)?([a-zA-Z0-9_-]*)`)
	valueRE := regexp.MustCompile(`(.*-? +["\[]?)?([a-zA-Z0-9_-]*)(["\]]?)`)

	// Separate markup/comment
	markup, comment, isComment := strings.Cut(line, "#")
	if isComment {
		comment = color.HiBlackString("#%s", comment)
	}

	// Separate key/value
	key, value, isMapping := strings.Cut(markup, ":")
	if isMapping {
		key = keyRE.ReplaceAllString(key, fmt.Sprintf("$1%s$3", color.YellowString("$2")))
		value = valueRE.ReplaceAllString(value, fmt.Sprintf("$1%s$3", color.CyanString("$2")))
		return fmt.Sprintf("%s:%s%s", key, value, comment)
	} else {
		value = valueRE.ReplaceAllString(key, fmt.Sprintf("$1%s$3", color.CyanString("$2")))
		return fmt.Sprintf("%s%s", value, comment)
	}
}

func digits(number int) int {
	count := 0
	for number != 0 {
		number /= 10
		count += 1
	}
	return count
}
