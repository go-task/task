package tui

import (
	"context"
	"io"
	"slices"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type terminalProbe struct {
	env     tea.EnvMsg
	profile *colorprofile.Profile
}

func (m terminalProbe) Init() tea.Cmd { return nil }

func (m terminalProbe) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.EnvMsg:
		m.env = msg
	case tea.ColorProfileMsg:
		profile := msg.Profile
		m.profile = &profile
	}
	if m.env != nil && m.profile != nil {
		return m, tea.Quit
	}
	return m, nil
}

func (m terminalProbe) View() tea.View { return tea.NewView("") }

func TestJediTermPreservesColoursAndEnvironment(t *testing.T) {
	t.Parallel()

	for _, extra := range []string{"COLORTERM=truecolor", "COLORTERM=", "NO_COLOR=1"} {
		t.Run(extra, func(t *testing.T) {
			t.Parallel()
			environ := []string{"TERM=xterm-256color", "TERMINAL_EMULATOR=JetBrains-JediTerm", "CLICOLOR_FORCE=1", extra}
			original := slices.Clone(environ)
			options := terminalOptions(io.Discard, environ)
			require.NotEmpty(t, options)
			assert.Equal(t, original, environ, "must not change the subprocess environment")
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			options = append(options, tea.WithContext(ctx), tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutSignals())
			final, err := tea.NewProgram(terminalProbe{}, options...).Run()
			require.NoError(t, err)
			probe := final.(terminalProbe)
			assert.Equal(t, colorprofile.Detect(io.Discard, original), *probe.profile)
			assert.Equal(t, "JetBrains-JediTerm", probe.env.Getenv("TERMINAL_EMULATOR"))
		})
	}
}

func TestTerminalWorkaroundIsLimitedToDirectJediTermSessions(t *testing.T) {
	t.Parallel()

	for name, env := range map[string][]string{
		"ubuntu": {"TERM=xterm-256color", "VTE_VERSION=6800"},
		"tmux":   {"TERM=tmux-256color", "TERMINAL_EMULATOR=JetBrains-JediTerm", "TMUX=/tmp/tmux"},
		"screen": {"TERM=screen", "TERMINAL_EMULATOR=JetBrains-JediTerm", "STY=123.session"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, terminalOptions(io.Discard, env))
		})
	}
}
