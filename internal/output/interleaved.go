package output

import (
	"io"
)

type Interleaved struct{}

func (Interleaved) WrapWriter(w io.Writer, _ string, _ Templater) io.Writer {
	return w
}
