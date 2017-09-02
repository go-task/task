// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

// Package interp implements an interpreter that executes shell
// programs. It aims to support POSIX, but its support is not complete
// yet. It also supports some Bash features.
//
// This package is a work in progress and EXPERIMENTAL; its API is not
// subject to the 1.x backwards compatibility guarantee.
package interp // import "mvdan.cc/sh/interp"
