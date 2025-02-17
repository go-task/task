package errors

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

type Snippet struct {
	lines   []string
	start   int
	end     int
	line    int
	column  int
	padding int
}

// NewSnippet creates a new snippet from a byte slice and a line and column
// number. The line and column numbers should be 1-indexed. For example, the
// first character in the file would be 1:1 (line 1, column 1). The padding
// determines the number of lines to include before and after the chosen line.
func NewSnippet(b []byte, line, column, padding int) *Snippet {
	line = max(line, 1)
	column = max(column, 1)

	// Syntax highlight the snippet
	buf := &bytes.Buffer{}
	if err := quick.Highlight(buf, string(b), "yaml", "terminal", "task"); err != nil {
		buf.WriteString(string(b))
	}

	// Work out the start and end lines of the snippet
	lines := strings.Split(buf.String(), "\n")
	start := max(line-padding, 1)
	end := min(line+padding, len(lines)-1)

	// Return the snippet
	return &Snippet{
		lines:   lines[start-1 : end],
		start:   start,
		end:     end,
		line:    line,
		column:  column,
		padding: padding,
	}
}

func (snippet *Snippet) String() string {
	buf := &bytes.Buffer{}

	maxLineNumberDigits := digits(snippet.end)
	lineNumberFormat := fmt.Sprintf("%%%dd", maxLineNumberDigits)
	lineNumberSpacer := strings.Repeat(" ", maxLineNumberDigits)
	lineIndicatorSpacer := strings.Repeat(" ", len(lineIndicator))
	columnSpacer := strings.Repeat(" ", snippet.column-1)

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
