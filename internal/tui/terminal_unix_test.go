//go:build linux || darwin

package tui

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dashboardFrameMsg struct{ appModel }

type dashboardStartup struct {
	ready                      chan struct{}
	size, environment, profile bool
}

func (s *dashboardStartup) observe(msg tea.Msg) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		s.size = true
	case tea.EnvMsg:
		s.environment = true
	case tea.ColorProfileMsg:
		s.profile = true
	}
	if s.size && s.environment && s.profile && s.ready != nil {
		close(s.ready)
		s.ready = nil
	}
}

// dashboardFrames swaps one complete dashboard view in a single update. Its
// timestamps and content are fixed instead of depending on timed commands.
type dashboardFrames struct {
	appModel
	startup *dashboardStartup
}

func (m dashboardFrames) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if frame, ok := msg.(dashboardFrameMsg); ok {
		return dashboardFrames{appModel: frame.appModel, startup: m.startup}, nil
	}
	next, cmd := m.appModel.Update(msg)
	m.appModel = next.(appModel)
	m.startup.observe(msg)
	return m, cmd
}

func dashboardFixture(t *testing.T, notice string, next bool) appModel {
	t.Helper()
	m := newTUIModel(func() {})
	m = updateTUIModel(t, m, tea.WindowSizeMsg{Width: 90, Height: 28})
	at := time.Unix(1700000000, 0)
	for id := uint64(1); id <= 12; id++ {
		task := m.scheduleTask(taskInvocation{ID: id, RootID: id, Name: fmt.Sprintf("worker-%02d", id)})
		duration := 1100 * time.Millisecond
		task.state, task.startedAt, task.finishedAt = taskSucceeded, at, at.Add(duration)
	}
	m.done = true
	m.notice = notice
	// The two changed characters are on consecutive rows and straddle a fixed
	// tab stop in the output pane. This makes the control exercise Bubble Tea's
	// hard-tab optimization while keeping both frames deterministic.
	output := "a0000000\n0000000b\n"
	if next {
		output = "c0000000\n0000000d\n"
	}
	m.appendOutput(1, "worker-01", output)
	return newAppModel(launcherModel{}, m, false, nil, nil, nil)
}

func TestDashboardProgramWritesCompatibleCursorMoves(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, emulator, colour string
		compatible             bool
	}{
		{"control", "", "COLORTERM=truecolor", false},
		{"jediterm", "JetBrains-JediTerm", "COLORTERM=truecolor", true},
		{"256-colours", "JetBrains-JediTerm", "COLORTERM=", true},
		{"no-colour", "JetBrains-JediTerm", "NO_COLOR=1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			terminal := newTerminalPTY(t, 90, 28)
			environ := []string{"TERM=xterm-256color", "TERMINAL_EMULATOR=" + tc.emulator, tc.colour}
			original := slices.Clone(environ)
			ui := &UI{input: terminal.tty, output: terminal.tty}
			startupReady := make(chan struct{})
			startup := &dashboardStartup{ready: startupReady}
			program := ui.newProgram(dashboardFrames{
				appModel: dashboardFixture(t, "terminal-initial", false),
				startup:  startup,
			}, environ)
			done := make(chan error, 1)
			go func() { _, err := program.Run(); done <- err }()
			t.Cleanup(program.Kill)
			select {
			case <-startupReady:
			case <-time.After(5 * time.Second):
				t.Fatal("program did not finish terminal initialization")
			}
			initial := len(terminal.snapshot())
			program.Send(dashboardFrameMsg{dashboardFixture(t, "terminal-frame-one", false)})
			terminal.await(t, func(s string) bool {
				return len(s) > initial && strings.Contains(s[initial:], "frame-one")
			})
			before := len(terminal.snapshot())
			program.Send(dashboardFrameMsg{dashboardFixture(t, "terminal-frame-two", true)})
			terminal.await(t, func(s string) bool { return len(s) > before && strings.Contains(s[before:], "two") })
			program.Quit()
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("program did not quit")
			}
			written := terminal.snapshot()
			tabMoves := regexp.MustCompile(`\t|\x1b\[[0-9]*[IZ]`)
			if tc.compatible {
				assert.False(t, tabMoves.MatchString(written), "compatible program emitted HT, CHT, or CBT")
			} else {
				assert.True(t, tabMoves.MatchString(written), "control must exercise a terminal tab optimization")
			}
			assert.Equal(t, original, environ, "must not change the subprocess environment")
			switch tc.colour {
			case "COLORTERM=truecolor":
				assert.Contains(t, written, "38;2;")
			case "COLORTERM=":
				assert.Contains(t, written, "38;5;")
			case "NO_COLOR=1":
				assert.False(t, regexp.MustCompile(`\x1b\[[0-9;]*[34]8;`).MatchString(written))
			}
		})
	}
}
