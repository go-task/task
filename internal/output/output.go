package output

import (
	"io"
)


// Templater executes a template engine.
// It is provided by the templater.Templater package.
type Templater interface {
	// Replace replaces the provided template string with a rendered string.
	Replace(tmpl string) string
}

type Output interface {
	WrapWriter(w io.Writer, prefix string, tmpl Templater) io.Writer
}
