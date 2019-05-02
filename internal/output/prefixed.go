package output

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type Prefixed struct{}

func (Prefixed) WrapWriter(w io.Writer, prefix string) io.Writer {
	return &prefixWriter{writer: w, prefix: prefix}
}

type prefixWriter struct {
	writer io.Writer
	prefix string
	buff   bytes.Buffer
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	n, err := pw.buff.Write(p)
	if err != nil {
		return n, err
	}

	return n, pw.writeOutputLines(false)
}

func (pw *prefixWriter) Close() error {
	return pw.writeOutputLines(true)
}

func (pw *prefixWriter) writeOutputLines(force bool) error {
	for {
		switch line, err := pw.buff.ReadString('\n'); err {
		case nil:
			if err = pw.writeLine(line); err != nil {
				return err
			}
		case io.EOF:
			// if this line was not a complete line, re-add to the buffer
			if !force && !strings.HasSuffix(line, "\n") {
				_, err = pw.buff.WriteString(line)
				return err
			}

			return pw.writeLine(line)
		default:
			return err
		}
	}
}

func (pw *prefixWriter) writeLine(line string) error {
	if line == "" {
		return nil
	}
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	_, err := fmt.Fprintf(pw.writer, "[%s] %s", pw.prefix, line)
	return err
}
