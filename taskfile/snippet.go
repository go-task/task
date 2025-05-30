package taskfile

import (
	"bytes"
	"embed"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/quick"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/fatih/color"
)

//go:embed themes/*.xml
var embedded embed.FS

const (
	lineIndicator   = ">"
	columnIndicator = "^"
)

func init() {
	r, err := embedded.Open("themes/task.xml")
	if err != nil {
		panic(err)
	}
	style, err := chroma.NewXMLStyle(r)
	if err != nil {
		panic(err)
	}
	styles.Register(style)
}

type (
	// A SnippetOption is any type that can apply a configuration to a [Snippet].
	SnippetOption interface {
		ApplyToSnippet(*Snippet)
	}
	// A Snippet is a syntax highlighted snippet of a Taskfile with optional
	// padding and a line and column indicator.
	Snippet struct {
		linesRaw         []string
		linesHighlighted []string
		start            int
		end              int
		line             int
		column           int
		padding          int
		noIndicators     bool
	}
)

// NewSnippet creates a new [Snippet] from a byte slice and a line and column
// number. The line and column numbers should be 1-indexed. For example, the
// first character in the file would be 1:1 (line 1, column 1). The padding
// determines the number of lines to include before and after the chosen line.
func NewSnippet(b []byte, opts ...SnippetOption) *Snippet {
	snippet := &Snippet{}
	snippet.Options(opts...)

	// Syntax highlight the input and split it into lines
	buf := &bytes.Buffer{}
	if err := quick.Highlight(buf, string(b), "yaml", "terminal", "task"); err != nil {
		buf.Write(b)
	}
	linesRaw := strings.Split(string(b), "\n")
	linesHighlighted := strings.Split(buf.String(), "\n")

	// Work out the start and end lines of the snippet
	snippet.start = max(snippet.line-snippet.padding, 1)
	snippet.end = min(snippet.line+snippet.padding, len(linesRaw)-1)
	snippet.linesRaw = linesRaw[snippet.start-1 : snippet.end]
	snippet.linesHighlighted = linesHighlighted[snippet.start-1 : snippet.end]

	return snippet
}

// Options loops through the given [SnippetOption] functions and applies them
// to the [Snippet].
func (s *Snippet) Options(opts ...SnippetOption) {
	for _, opt := range opts {
		opt.ApplyToSnippet(s)
	}
}

// WithLine specifies the line number that the [Snippet] should center around
// and point to.
func WithLine(line int) SnippetOption {
	return &lineOption{line: line}
}

type lineOption struct {
	line int
}

func (o *lineOption) ApplyToSnippet(s *Snippet) {
	s.line = o.line
}

// WithColumn specifies the column number that the [Snippet] should point to.
func WithColumn(column int) SnippetOption {
	return &columnOption{column: column}
}

type columnOption struct {
	column int
}

func (o *columnOption) ApplyToSnippet(s *Snippet) {
	s.column = o.column
}

// WithPadding specifies the number of lines to include before and after the
// selected line in the [Snippet].
func WithPadding(padding int) SnippetOption {
	return &paddingOption{padding: padding}
}

type paddingOption struct {
	padding int
}

func (o *paddingOption) ApplyToSnippet(s *Snippet) {
	s.padding = o.padding
}

// WithNoIndicators specifies that the [Snippet] should not include line or
// column indicators.
func WithNoIndicators() SnippetOption {
	return &noIndicatorsOption{}
}

type noIndicatorsOption struct{}

func (o *noIndicatorsOption) ApplyToSnippet(s *Snippet) {
	s.noIndicators = true
}

func (s *Snippet) String() string {
	buf := &bytes.Buffer{}

	maxLineNumberDigits := digits(s.end)
	lineNumberFormat := fmt.Sprintf("%%%dd", maxLineNumberDigits)
	lineNumberSpacer := strings.Repeat(" ", maxLineNumberDigits)
	lineIndicatorSpacer := strings.Repeat(" ", len(lineIndicator))
	columnSpacer := strings.Repeat(" ", max(s.column-1, 0))

	// Loop over each line in the snippet
	for i, lineHighlighted := range s.linesHighlighted {
		if i > 0 {
			fmt.Fprintln(buf)
		}

		currentLine := s.start + i
		lineNumber := fmt.Sprintf(lineNumberFormat, currentLine)

		// If this is a padding line or indicators are disabled, print it as normal
		if currentLine != s.line || s.noIndicators {
			fmt.Fprintf(buf, "%s %s | %s", lineIndicatorSpacer, lineNumber, lineHighlighted)
			continue
		}

		// Otherwise, print the line with indicators
		fmt.Fprintf(buf, "%s %s | %s", color.RedString(lineIndicator), lineNumber, lineHighlighted)

		// Only print the column indicator if the column is in bounds
		if s.column > 0 && s.column <= len(s.linesRaw[i]) {
			fmt.Fprintf(buf, "\n%s %s | %s%s", lineIndicatorSpacer, lineNumberSpacer, columnSpacer, color.RedString(columnIndicator))
		}
	}

	// If there are lines, but no line is selected, print the column indicator under all the lines
	if len(s.linesHighlighted) > 0 && s.line == 0 && s.column > 0 {
		fmt.Fprintf(buf, "\n%s %s | %s%s", lineIndicatorSpacer, lineNumberSpacer, columnSpacer, color.RedString(columnIndicator))
	}

	return buf.String()
}

func digits(number int) int {
	count := 0
	for number != 0 {
		number /= 10
		count += 1
	}
	return count
}
