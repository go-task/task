//go:build windows
// +build windows

package main

import (
	"os"
	"runtime"

	"golang.org/x/sys/windows"
)

func init() {
	// Ensure that Windows console handles ANSI escape-codes correctly.
	if runtime.GOOS == "windows" {
		stdout := windows.Handle(os.Stdout.Fd())
		var originalMode uint32
		if err := windows.GetConsoleMode(stdout, &originalMode); err == nil {
			windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
		}
	}
}
