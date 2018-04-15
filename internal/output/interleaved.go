package output

import (
	"io"
)

type Interleaved struct{}

func (Interleaved) WrapWriter(w io.Writer) io.WriteCloser {
	return nopWriterCloser{w: w}
}

type nopWriterCloser struct {
	w io.Writer
}

func (wc nopWriterCloser) Write(p []byte) (int, error) {
	return wc.w.Write(p)
}

func (wc nopWriterCloser) Close() error {
	return nil
}
