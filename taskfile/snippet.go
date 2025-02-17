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
	SnippetOption func(*Snippet)
	Snippet       struct {
		lines   []string
		start   int
		end     int
		line    int
		column  int
		padding int
	}
)

// NewSnippet creates a new snippet from a byte slice and a line and column
// number. The line and column numbers should be 1-indexed. For example, the
// first character in the file would be 1:1 (line 1, column 1). The padding
// determines the number of lines to include before and after the chosen line.
func NewSnippet(b []byte, opts ...SnippetOption) *Snippet {
	snippet := &Snippet{}
	for _, opt := range opts {
		opt(snippet)
	}

	// Syntax highlight the input and split it into lines
	buf := &bytes.Buffer{}
	if err := quick.Highlight(buf, string(b), "yaml", "terminal", "task"); err != nil {
		buf.WriteString(string(b))
	}
	lines := strings.Split(buf.String(), "\n")

	// Work out the start and end lines of the snippet
	snippet.start = max(snippet.line-snippet.padding, 1)
	snippet.end = min(snippet.line+snippet.padding, len(lines)-1)
	snippet.lines = lines[snippet.start-1 : snippet.end]

	return snippet
}

func SnippetWithLine(line int) SnippetOption {
	return func(snippet *Snippet) {
		snippet.line = line
	}
}

func SnippetWithColumn(column int) SnippetOption {
	return func(snippet *Snippet) {
		snippet.column = column
	}
}

func SnippetWithPadding(padding int) SnippetOption {
	return func(snippet *Snippet) {
		snippet.padding = padding
	}
}

func (snippet *Snippet) String() string {
	buf := &bytes.Buffer{}

	maxLineNumberDigits := digits(snippet.end)
	lineNumberFormat := fmt.Sprintf("%%%dd", maxLineNumberDigits)
	lineNumberSpacer := strings.Repeat(" ", maxLineNumberDigits)
	lineIndicatorSpacer := strings.Repeat(" ", len(lineIndicator))
	columnSpacer := strings.Repeat(" ", max(snippet.column-1, 0))

	// Loop over each line in the snippet
	for i, line := range snippet.lines {
		if i > 0 {
			fmt.Fprintln(buf)
		}

		currentLine := snippet.start + i
		lineNumber := fmt.Sprintf(lineNumberFormat, currentLine)

		// If this is a padding line, print it as normal
		if currentLine != snippet.line {
			fmt.Fprintf(buf, "%s %s | %s", lineIndicatorSpacer, lineNumber, line)
			continue
		}

		// Otherwise, print the line with indicators
		fmt.Fprintf(buf, "%s %s | %s", color.RedString(lineIndicator), lineNumber, line)
		if snippet.column > 0 {
			fmt.Fprintf(buf, "\n%s %s | %s%s", lineIndicatorSpacer, lineNumberSpacer, columnSpacer, color.RedString(columnIndicator))
		}
	}

	// If there are lines, but no line is selected, print the column indicator under all the lines
	if len(snippet.lines) > 0 && snippet.line == 0 && snippet.column > 0 {
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
