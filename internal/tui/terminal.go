package tui

import (
	"io"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
)

// terminalOptions avoids backward-tab cursor movements on JediTerm, where
// they can leave incremental duration updates in the wrong pane.
// https://github.com/charmbracelet/bubbletea/pull/1641
// Bubble Tea does not yet expose individual renderer capabilities. The linux
// capability set omits backward tabs and hard tabs. Only the UI sees this TERM;
// subprocesses keep their environment, and colours use the original profile.
func terminalOptions(output io.Writer, environ []string) []tea.ProgramOption {
	env := uv.Environ(environ)
	if env.Getenv("TERMINAL_EMULATOR") != "JetBrains-JediTerm" {
		return nil
	}
	// A multiplexer owns cursor movement even when started inside an IDE.
	if env.Getenv("TMUX") != "" || env.Getenv("STY") != "" {
		return nil
	}
	profile := colorprofile.Detect(output, environ)
	compatible := slices.DeleteFunc(slices.Clone(environ), func(value string) bool {
		return strings.HasPrefix(value, "TERM=")
	})
	compatible = append(compatible, "TERM=linux")
	return []tea.ProgramOption{
		tea.WithEnvironment(compatible),
		tea.WithColorProfile(profile),
	}
}
