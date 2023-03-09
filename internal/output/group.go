package output

import (
	"bytes"
	"io"
)

type Group struct {
	Begin, End string
	ErrorOnly  bool
}

func (g Group) WrapWriter(stdOut, _ io.Writer, _ string, tmpl Templater) (io.Writer, io.Writer, CloseFunc) {
	gw := &groupWriter{writer: stdOut}
	if g.Begin != "" {
		gw.begin = tmpl.Replace(g.Begin) + "\n"
	}
	if g.End != "" {
		gw.end = tmpl.Replace(g.End) + "\n"
	}
	return gw, gw, func(err error) error {
		if g.ErrorOnly && err == nil {
			return nil
		}

		return gw.close()
	}
}

type groupWriter struct {
	writer     io.Writer
	buff       bytes.Buffer
	begin, end string
}

func (gw *groupWriter) Write(p []byte) (int, error) {
	return gw.buff.Write(p)
}

func (gw *groupWriter) close() error {
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
