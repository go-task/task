package output

import (
	"bytes"
	"io"
)

type Group struct{
	Begin, End string
}

func (g Group) WrapWriter(w io.Writer, _ string, tmpl Templater) io.Writer {
	gw := &groupWriter{writer: w}
	if g.Begin != "" {
		gw.begin = tmpl.Replace(g.Begin) + "\n"
	}
	if g.End != "" {
		gw.end = tmpl.Replace(g.End) + "\n"
	}
	return gw
}

type groupWriter struct {
	writer io.Writer
	buff   bytes.Buffer
	begin, end string
}

func (gw *groupWriter) Write(p []byte) (int, error) {
	return gw.buff.Write(p)
}

func (gw *groupWriter) Close() error {
	if gw.buff.Len() == 0 {
		// don't print begin/end messages if there's no buffered entries
		return nil
	}
	if _, err := io.WriteString(gw.writer, gw.begin); err != nil {
		return err
	}
	gw.buff.WriteString(gw.end)
	_, err := io.Copy(gw.writer, &gw.buff)
	return err
}
