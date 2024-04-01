package output

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
)

type Prefixed struct {
	logger  *logger.Logger
	seen    map[string]uint
	counter *uint
	color   bool
}

func NewPrefixed(logger *logger.Logger, color bool) Prefixed {
	var counter uint

	return Prefixed{
		seen:    make(map[string]uint),
		counter: &counter,
		logger:  logger,
		color:   color,
	}
}

func (p Prefixed) WrapWriter(stdOut, _ io.Writer, prefix string, _ *templater.Cache) (io.Writer, io.Writer, CloseFunc) {
	pw := &prefixWriter{writer: stdOut, prefix: prefix, prefixed: &p}
	return pw, pw, func(error) error { return pw.close() }
}

type prefixWriter struct {
	writer   io.Writer
	prefixed *Prefixed
	prefix   string
	buff     bytes.Buffer
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

var PrefixColorSequence = []logger.Color{
	logger.Yellow, logger.Blue, logger.Magenta, logger.Cyan, logger.Green, logger.Red,
	// bYellow, bBlue, bMagenta, bCyan, bGreen, bRed,
}

func (pw *prefixWriter) writeLine(line string) error {
	if line == "" {
		return nil
	}
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}

	idx, ok := pw.prefixed.seen[pw.prefix]

	if !ok {
		idx = *pw.prefixed.counter
		pw.prefixed.seen[pw.prefix] = idx

		*pw.prefixed.counter++
	}

	if _, err := fmt.Fprint(pw.writer, "["); err != nil {
		return nil
	}

	if pw.prefixed.color {
		color := PrefixColorSequence[idx%uint(len(PrefixColorSequence))]
		pw.prefixed.logger.FOutf(pw.writer, color, pw.prefix)
	} else {
		if _, err := fmt.Fprint(pw.writer, pw.prefix); err != nil {
			return nil
		}
	}

	if _, err := fmt.Fprint(pw.writer, "] "); err != nil {
		return nil
	}

	_, err := fmt.Fprint(pw.writer, line)
	return err
}
