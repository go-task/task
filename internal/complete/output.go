package complete

import (
	"fmt"
	"io"
	"strings"
)

// Write emits the cobra-v2 completion protocol: one `value\tdescription` (or
// bare `value`) per suggestion, followed by a trailing `:<directive>` line
// that shell wrappers split off even when there are zero suggestions.
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

func sanitize(s string) string {
	r := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ")
	return r.Replace(s)
}
