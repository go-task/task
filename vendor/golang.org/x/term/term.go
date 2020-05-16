// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package term provides support functions for dealing with terminals, as
// commonly found on UNIX systems.
package term

// IsTerminal returns whether the given file descriptor is a terminal.
func IsTerminal(fd int) bool {
	return isTerminal(fd)
}
