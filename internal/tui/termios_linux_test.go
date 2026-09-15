//go:build linux

package tui

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func enableTerminalTabStops(t *testing.T, tty *os.File) {
	t.Helper()
	state, err := unix.IoctlGetTermios(int(tty.Fd()), unix.TCGETS)
	require.NoError(t, err)
	state.Oflag &^= unix.TABDLY
	require.NoError(t, unix.IoctlSetTermios(int(tty.Fd()), unix.TCSETS, state))
}
