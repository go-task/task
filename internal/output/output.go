package output

import (
	"io"
)

type Output interface {
	WrapWriter(w io.Writer, prefix string) io.Writer
}
