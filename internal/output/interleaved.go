package output

import (
	"io"
)

type Interleaved struct{}

func (Interleaved) WrapWriter(w io.Writer, _ string) io.Writer {
	return w
}
