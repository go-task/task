//go:build linux || darwin

package tui

import (
	"bytes"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/require"
)

// terminalPTY records what the terminal receives, rather than asking a second
// renderer to reproduce what we think Bubble Tea should write.
type terminalPTY struct {
	master, tty *os.File
	mu          sync.Mutex
	output      bytes.Buffer
	changed     chan struct{}
}

func newTerminalPTY(t *testing.T, width, height uint16) *terminalPTY {
	t.Helper()
	master, tty, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { _ = tty.Close(); _ = master.Close() })
	// Bubble Tea reads this input termios state before entering raw mode to
	// decide whether the renderer may use terminal tab stops.
	enableTerminalTabStops(t, tty)
	require.NoError(t, pty.Setsize(tty, &pty.Winsize{Cols: width, Rows: height}))
	p := &terminalPTY{master: master, tty: tty, changed: make(chan struct{}, 1)}
	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := master.Read(buf)
			if n > 0 {
				p.mu.Lock()
				_, _ = p.output.Write(buf[:n])
				p.mu.Unlock()
				select {
				case p.changed <- struct{}{}:
				default:
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return p
}

func (p *terminalPTY) snapshot() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.output.String()
}

func (p *terminalPTY) await(t *testing.T, condition func(string) bool) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for !condition(p.snapshot()) {
		select {
		case <-p.changed:
		case <-timer.C:
			t.Fatal("timed out waiting for terminal output")
		}
	}
}
