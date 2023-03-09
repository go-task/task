package output

import (
	"io"
)

type Interleaved struct{}

func (Interleaved) WrapWriter(stdOut, stdErr io.Writer, _ string, _ Templater) (io.Writer, io.Writer, CloseFunc) {
	return stdOut, stdErr, func(error) error { return nil }
}
