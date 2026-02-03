package output

import (
	"bytes"
	"io"

	"github.com/go-task/task/v3/internal/templater"
)

type Group struct {
	Begin, End string
	ErrorOnly  bool
}

func (g Group) WrapWriter(stdOut, _ io.Writer, _ string, cache *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	gw := &groupWriter{writer: stdOut}
	if g.Begin != "" {
		gw.begin = templater.Replace(g.Begin, cache) + "\n"
	}
	if g.End != "" {
		gw.end = templater.Replace(g.End, cache) + "\n"
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
	switch {
	case gw.buff.Len() == 0:
		return nil
	case gw.begin == "" && gw.end == "":
		_, err := io.Copy(gw.writer, &gw.buff)
		return err
	default:
		_, err := io.Copy(gw.writer, gw.combinedBuff())
		return err
	}
}

func (gw *groupWriter) combinedBuff() io.Reader {
	var b bytes.Buffer
	_, _ = b.WriteString(gw.begin)
	_, _ = io.Copy(&b, &gw.buff)
	_, _ = b.WriteString(gw.end)
	return &b
}
