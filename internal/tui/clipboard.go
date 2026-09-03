package tui

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// clipboardTimeout bounds a clipboard helper that misbehaves. Copying is not
// worth hanging the interface over.
const clipboardTimeout = 3 * time.Second

type clipboardCopiedMsg struct {
	size      int
	confirmed bool // a system clipboard tool accepted the text
}

// systemClipboardArgs returns the command that puts stdin on the clipboard, or
// false when this machine has none.
//
// OSC 52 alone is not enough. VTE, which backs GNOME Terminal and the other
// Ubuntu terminals, does not implement it and swallows the sequence without
// error, so a helper is the only thing that works there.
func systemClipboardArgs() ([]string, bool) {
	candidates := [][]string{}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		candidates = append(candidates, []string{"wl-copy"})
	}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, []string{"pbcopy"})
	}
	if os.Getenv("DISPLAY") != "" {
		candidates = append(candidates,
			[]string{"xclip", "-selection", "clipboard"},
			[]string{"xsel", "--clipboard", "--input"},
		)
	}
	// Also covers WSL, where the Windows helper is on PATH.
	candidates = append(candidates, []string{"clip.exe"})

	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate[0]); err == nil {
			return candidate, true
		}
	}
	return nil, false
}

// copyToSystemClipboard runs the clipboard helper, if there is one. It reports
// whether the text was definitely copied, which OSC 52 can never tell us.
func copyToSystemClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		args, ok := systemClipboardArgs()
		if !ok {
			return clipboardCopiedMsg{size: len(text)}
		}

		ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
		defer cancel()
		// args comes from the fixed candidate list above, never from the
		// Taskfile or the user, and the text goes in over stdin.
		cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec
		cmd.Stdin = strings.NewReader(text)
		// Leave the helper's output attached to nothing. Helpers such as xclip
		// and wl-copy fork to hold the selection, and inheriting a pipe would
		// keep us waiting for that background process to exit.
		cmd.Stdout, cmd.Stderr = nil, nil

		if err := cmd.Run(); err != nil {
			return clipboardCopiedMsg{size: len(text)}
		}
		return clipboardCopiedMsg{size: len(text), confirmed: true}
	}
}
