package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/logger"
)

func TestParallelTasksDoNotWaitForTUI(t *testing.T) {
	t.Parallel()
	for _, paused := range []bool{true, false} {
		t.Run(fmt.Sprintf("paused=%t", paused), func(t *testing.T) {
			t.Parallel()
			testParallelTasksWithTUI(t, paused)
		})
	}
}

func testParallelTasksWithTUI(t *testing.T, paused bool) {
	t.Helper()

	const workers = 32
	e := parallelTaskExecutor(t, workers)
	ui, err := New(&logger.Logger{AssumeTerm: true}, Options{})
	require.NoError(t, err)
	execution := newTUIModel(func() {})
	execution.quitting = true // Exit as soon as the queued completion arrives.
	app := newAppModel(launcherModel{}, execution, false, nil, nil, nil)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	program := tea.NewProgram(app, tea.WithContext(ctx), tea.WithInput(nil),
		tea.WithOutput(io.Discard), tea.WithoutRenderer(), tea.WithoutSignals())
	ui.program = program
	e.Listener = ui.listener()
	type programResult struct {
		model tea.Model
		err   error
	}
	result := make(chan programResult, 1)
	runProgram := func() {
		model, err := program.Run()
		result <- programResult{model: model, err: err}
	}
	if !paused {
		go runProgram()
	}

	// In the paused case, no UI consumer exists yet. Synchronous Program.Send
	// would prevent the first Scheduled callback from returning.
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
		program.Kill()
		t.Fatal("task execution waited for the UI")
	}

	if paused {
		go runProgram()
	}
	final := <-result
	require.NoError(t, final.err)
	m := final.model.(appModel).execution
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
	assert.False(t, ui.send(taskScheduledMsg{}), "events after shutdown must be dropped")
}

func parallelTaskExecutor(t *testing.T, workers int) *task.Executor {
	t.Helper()
	dir := t.TempDir()
	var taskfile strings.Builder
	taskfile.WriteString("version: '3'\ntasks:\n  all:\n    deps:\n")
	for i := range workers {
		fmt.Fprintf(&taskfile, "      - worker-%d\n", i)
	}
	for i := range workers {
		fmt.Fprintf(&taskfile, "  worker-%d:\n    cmds: ['echo done-%d']\n", i, i)
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile.String()), 0o600))
	e := task.NewExecutor(task.WithDir(dir), task.WithStdout(io.Discard), task.WithStderr(io.Discard))
	require.NoError(t, e.Setup())
	return e
}

func TestAppDoesNotDispatchNestedQueueWakeups(t *testing.T) {
	t.Parallel()
	nested := &UI{messages: []tea.Msg{started(1, 0, "nested")}}
	queue := &UI{messages: []tea.Msg{messagesReadyMsg{ui: nested}, started(2, 0, "ordinary")}}
	app := newAppModel(launcherModel{}, newTUIModel(func() {}), false, nil, nil, nil)
	next, _ := app.Update(messagesReadyMsg{ui: queue})
	assert.Nil(t, next.(appModel).execution.byID[1])
	assert.NotNil(t, next.(appModel).execution.byID[2])
	assert.Len(t, nested.messages, 1, "a notification inside a batch is not another batch to drain")
}
