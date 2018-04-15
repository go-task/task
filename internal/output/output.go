package output

import (
	"io"
)

type Output interface {
	WrapWriter(io.Writer) io.WriteCloser
}
