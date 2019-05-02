package output

import (
	"bytes"
	"io"
)

type Group struct{}

func (Group) WrapWriter(w io.Writer, _ string) io.Writer {
	return &groupWriter{writer: w}
}

type groupWriter struct {
	writer io.Writer
	buff   bytes.Buffer
}

func (gw *groupWriter) Write(p []byte) (int, error) {
	return gw.buff.Write(p)
}

func (gw *groupWriter) Close() error {
	_, err := io.Copy(gw.writer, &gw.buff)
	return err
}
