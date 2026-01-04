package output

import (
	"bytes"
	"io"

	"github.com/go-task/task/v3/internal/templater"
)

type Logger struct{}

func (l Logger) WrapWriter(stdOut, stdErr io.Writer, _ string, cache *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	lwOut := &LoggerWriter{writer: stdOut}
	lwErr := &LoggerWriter{writer: stdErr}
	return lwOut, lwErr, func(error) error { return nil }
}

type LoggerWriter struct {
	writer io.Writer
	Buffer bytes.Buffer
}

func (lw *LoggerWriter) Write(p []byte) (int, error) {
	return lw.Buffer.Write(p)
}
