package output

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/go-task/task/v3/internal/templater"
)

type Prefixed struct {
	seen    map[string]uint
	counter *uint
	Color   bool
}

func NewPrefixed(color bool) Prefixed {
	var counter uint

	return Prefixed{
		Color:   color,
		counter: &counter,
		seen:    make(map[string]uint),
	}
}

func (p Prefixed) WrapWriter(stdOut, _ io.Writer, prefix string, _ *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	pw := &prefixWriter{writer: stdOut, prefix: prefix, color: p.Color, seen: p.seen, counter: p.counter}
	return pw, pw, func(error) error { return pw.close() }
}

type prefixWriter struct {
	writer  io.Writer
	seen    map[string]uint
	counter *uint
	prefix  string
	buff    bytes.Buffer
	color   bool
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	n, err := pw.buff.Write(p)
	if err != nil {
		return n, err
	}

	return n, pw.writeOutputLines(false)
}

func (pw *prefixWriter) close() error {
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

var PrefixColorSequence = [][]byte{
	nYellow, nBlue, nMagenta, nCyan, nGreen, nRed,
	bYellow, bBlue, bMagenta, bCyan, bGreen, bRed,
}

func (pw *prefixWriter) writeLine(line string) error {
	if line == "" {
		return nil
	}
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}

	idx, ok := pw.seen[pw.prefix]

	if !ok {
		idx = *pw.counter
		pw.seen[pw.prefix] = idx

		*pw.counter += 1
	}

	color := PrefixColorSequence[idx%uint(len(PrefixColorSequence))]

	if _, err := fmt.Fprint(pw.writer, "["); err != nil {
		return nil
	}

	if err := cW(pw.writer, pw.color, color, "%s", pw.prefix); err != nil {
		return err
	}

	if _, err := fmt.Fprint(pw.writer, "] "); err != nil {
		return nil
	}

	_, err := fmt.Fprint(pw.writer, line)
	return err
}
