/*
Source code in this file is based on the code from the Go-Chi project.
https://github.com/go-chi/chi/blob/master/middleware/terminal.go

Copyright (c) 2015-present Peter Kieltyka (https://github.com/pkieltyka), Google Inc.

MIT License

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package output

import (
	"fmt"
	"io"
	"os"
)

var (
	// Normal colors
	// nBlack   = []byte{'\033', '[', '3', '0', 'm'} UNUSED
	nRed     = []byte{'\033', '[', '3', '1', 'm'}
	nGreen   = []byte{'\033', '[', '3', '2', 'm'}
	nYellow  = []byte{'\033', '[', '3', '3', 'm'}
	nBlue    = []byte{'\033', '[', '3', '4', 'm'}
	nMagenta = []byte{'\033', '[', '3', '5', 'm'}
	nCyan    = []byte{'\033', '[', '3', '6', 'm'}
	// nWhite   = []byte{'\033', '[', '3', '7', 'm'} UNUSED
	// Bright colors
	// bBlack   = []byte{'\033', '[', '3', '0', ';', '1', 'm'} UNUSED
	bRed     = []byte{'\033', '[', '3', '1', ';', '1', 'm'}
	bGreen   = []byte{'\033', '[', '3', '2', ';', '1', 'm'}
	bYellow  = []byte{'\033', '[', '3', '3', ';', '1', 'm'}
	bBlue    = []byte{'\033', '[', '3', '4', ';', '1', 'm'}
	bMagenta = []byte{'\033', '[', '3', '5', ';', '1', 'm'}
	bCyan    = []byte{'\033', '[', '3', '6', ';', '1', 'm'}
	// bWhite   = []byte{'\033', '[', '3', '7', ';', '1', 'm'} UNUSED

	reset = []byte{'\033', '[', '0', 'm'}
)

// This is public so we can override it in tests.
var IsTTY bool

func init() {
	// This is sort of cheating: if stdout is a character device, we assume
	// that means it's a TTY. Unfortunately, there are many non-TTY
	// character devices, but fortunately stdout is rarely set to any of
	// them.
	//
	// We could solve this properly by pulling in a dependency on
	// code.google.com/p/go.crypto/ssh/terminal, for instance, but as a
	// heuristic for whether to print in color or in black-and-white, I'd
	// really rather not.
	fi, err := os.Stdout.Stat()
	if err == nil {
		m := os.ModeDevice | os.ModeCharDevice
		IsTTY = fi.Mode()&m == m
	}
}

// colorWrite
func cW(w io.Writer, useColor bool, color []byte, s string, args ...any) error {
	if IsTTY && useColor {
		if _, err := w.Write(color); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, s, args...); err != nil {
		return err
	}

	if IsTTY && useColor {
		if _, err := w.Write(reset); err != nil {
			return err
		}
	}

	return nil
}
