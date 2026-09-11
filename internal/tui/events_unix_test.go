//go:build linux || darwin

package tui

import (
	"context"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/logger"
)

// blockedTTY preserves Fd so Bubble Tea still detects the real terminal. Once
// armed, an actual renderer Write stays blocked until the test releases it.
type blockedTTY struct {
	*os.File
	armed, entered, release chan struct{}
	once                    sync.Once
}

func (w *blockedTTY) Write(p []byte) (int, error) {
	select {
	case <-w.armed:
		w.once.Do(func() { close(w.entered) })
		<-w.release
	default:
	}
	return w.File.Write(p)
}

func TestParallelTasksFinishWhileTerminalWriteIsBlocked(t *testing.T) {
	t.Parallel()
	const workers = 32
	e := parallelTaskExecutor(t, workers)
	terminal := newTerminalPTY(t, 120, 40)
	writer := &blockedTTY{File: terminal.tty, armed: make(chan struct{}), entered: make(chan struct{}), release: make(chan struct{})}
	ui, err := New(&logger.Logger{AssumeTerm: true, Stdin: terminal.tty, Stdout: writer}, Options{})
	require.NoError(t, err)
	m := newTUIModel(func() {})
	m.quitting = true // Quit when the queued executionDoneMsg is processed.
	program := ui.newProgram(newAppModel(launcherModel{}, m, false, nil, nil, nil), []string{"TERM=xterm-256color"})
	ui.program = program
	e.Listener = ui.listener()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(writer.release) }) }
	t.Cleanup(func() { release(); program.Kill() })
	type programResult struct {
		model tea.Model
		err   error
	}
	result := make(chan programResult, 1)
	go func() { model, err := program.Run(); result <- programResult{model, err} }()
	terminal.await(t, func(s string) bool { return strings.Contains(s, "waiting for processes to exit") })

	close(writer.armed)
	go program.Send(tea.KeyPressMsg{Code: '?'}) // A changed view triggers the gated Write.
	select {
	case <-writer.entered:
	case <-ctx.Done():
		t.Fatal("renderer never reached the blocked terminal writer")
	}

	// At most one Send can be received before render waits for the renderer
	// lock. A startup event may already be holding up the loop, so do not require
	// the first probe to be received. The pair cannot finish until writes resume.
	probeStarted, probesReceived := make(chan struct{}), make(chan struct{})
	go func() {
		close(probeStarted)
		program.Send(elapsedTickMsg{})
		program.Send(elapsedTickMsg{})
		close(probesReceived)
	}()
	select {
	case <-probeStarted:
	case <-ctx.Done():
		t.Fatal("backpressure probe did not start")
	}
	select {
	case <-probesReceived:
		t.Fatal("blocked terminal did not produce Program.Send backpressure")
	case <-time.After(50 * time.Millisecond):
	}

	finished := make(chan error, 1)
	go func() {
		err := e.Run(ctx, &task.Call{Task: "all"})
		ui.send(executionDoneMsg{ui: ui, err: err})
		finished <- err
	}()
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("task execution waited for the blocked renderer")
	}

	// The UI cannot receive this batch yet. Check completion follows every
	// Finished event, and per-task ordering survives concurrent producers.
	ui.mutex.Lock()
	pending := slices.Clone(ui.messages)
	ui.mutex.Unlock()
	require.NotEmpty(t, pending)
	require.IsType(t, executionDoneMsg{}, pending[len(pending)-1])
	startedIDs, finishedIDs := map[uint64]bool{}, map[uint64]bool{}
	outputNotifications := 0
	for i, event := range pending {
		switch event := event.(type) {
		case taskStartedMsg:
			startedIDs[event.task.ID] = true
		case taskFinishedMsg:
			assert.True(t, startedIDs[event.id], "Finished before Started for %d", event.id)
			assert.False(t, finishedIDs[event.id], "duplicate Finished for %d", event.id)
			finishedIDs[event.id] = true
		case executionDoneMsg:
			assert.Equal(t, len(pending)-1, i, "completion must be the final event")
			assert.Len(t, finishedIDs, workers+1)
		case outputReadyMsg:
			outputNotifications++
		}
	}
	assert.Equal(t, 1, outputNotifications, "writes must coalesce while the UI is blocked")

	release()
	select {
	case final := <-result:
		require.NoError(t, final.err)
		m = final.model.(appModel).execution
	case <-ctx.Done():
		t.Fatal("program did not finish after releasing terminal writes")
	}
	require.True(t, m.done)
	require.Len(t, m.tasks, workers+1)
	for _, task := range m.tasks {
		assert.Equal(t, taskSucceeded, task.state, task.name)
		assert.False(t, task.startedAt.IsZero(), task.name)
		assert.False(t, task.finishedAt.IsZero(), task.name)
		if task.name != "all" {
			assert.Contains(t, task.output, "done-"+strings.TrimPrefix(task.name, "worker-"))
		}
	}
	assert.Empty(t, ui.drainMessages())
	assert.Empty(t, ui.drainOutput())
	close(ui.programDone)
}
