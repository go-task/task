package output

import (
	"io"

	"github.com/go-task/task/v3/internal/templater"
)

type Interleaved struct{}

func (Interleaved) WrapWriter(stdOut, stdErr io.Writer, _ string, _ *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	return stdOut, stdErr, func(error) error { return nil }
}
