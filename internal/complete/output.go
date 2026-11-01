package complete

import (
	"fmt"
	"io"
	"strings"
)

// The trailing `:<directive>` line is emitted even with zero suggestions.
func Write(w io.Writer, suggs []Suggestion, dir Directive) {
	for _, s := range suggs {
		value := sanitize(s.Value)
		desc := sanitize(s.Description)
		if desc == "" {
			fmt.Fprintln(w, value)
			continue
		}
		fmt.Fprintf(w, "%s\t%s\n", value, desc)
	}
	fmt.Fprintf(w, ":%d\n", dir)
}

// A value's tab or newline would be read as a field or record separator.
var completionSanitizer = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ")

func sanitize(s string) string {
	return completionSanitizer.Replace(s)
}
