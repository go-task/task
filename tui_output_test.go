package task_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

func TestTUIOutputStyleKeepsTaskfileOptions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
output:
  tui:
    status: icons
    task_navigator: tree
tasks:
  default: echo done
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o600))

	tests := []struct {
		name   string
		output ast.Output
		want   ast.OutputTUI
	}{
		{
			name:   "CLI selects TUI mode",
			output: ast.Output{Name: "tui"},
			want:   ast.OutputTUI{Status: "icons", TaskNavigator: "tree"},
		},
		{
			name:   "CLI option overrides one Taskfile option",
			output: ast.Output{Name: "tui", TUI: ast.OutputTUI{Status: "labels"}},
			want:   ast.OutputTUI{Status: "labels", TaskNavigator: "tree"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := task.NewExecutor(
				task.WithDir(dir),
				task.WithStdout(io.Discard),
				task.WithStderr(io.Discard),
				task.WithOutputStyle(test.output),
				task.WithAssumeTerm(true),
			)
			require.NoError(t, e.Setup())
			assert.Equal(t, test.want, e.OutputStyle.TUI)
		})
	}
}

func TestTaskLifecycleOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  default:
    deps: [first, second]
    cmds:
      - task: third
      - echo parent
  first:
    deps: [shared]
    cmds: [echo first]
  second:
    deps: [shared]
    cmds: [echo second]
  third: echo third
  shared:
    run: once
    cmds: [echo shared]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o600))

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
		task.WithSilent(true),
		task.WithForce(true),
	)
	require.NoError(t, e.Setup())
	recorder := &lifecycleRecorder{}
	e.Output = recorder

	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "default"}))
	require.Len(t, recorder.scheduled, 6)
	require.Len(t, recorder.started, 5)
	byName := make(map[string]output.TaskInvocation)
	for _, invocation := range recorder.started {
		byName[invocation.Name] = invocation
	}
	assert.Equal(t, 2, countInvocations(recorder.scheduled, "shared"))
	root := byName["default"]
	require.Len(t, recorder.joined, 1)
	for _, ownerID := range recorder.joined {
		assert.Equal(t, byName["shared"].ID, ownerID)
	}
	assert.Equal(t, root.ID, root.RootID)
	assert.Zero(t, root.ParentID)
	for _, name := range []string{"first", "second", "third"} {
		assert.Equal(t, root.ID, byName[name].ParentID, name)
	}
	for _, name := range []string{"first", "second", "third", "shared"} {
		assert.Equal(t, root.ID, byName[name].RootID, name)
	}
	var sharedParentIDs []uint64
	for _, invocation := range recorder.scheduled {
		if invocation.Name == "shared" {
			sharedParentIDs = append(sharedParentIDs, invocation.ParentID)
		}
	}
	assert.ElementsMatch(t, []uint64{byName["first"].ID, byName["second"].ID}, sharedParentIDs)
	scheduledIDs := make([]uint64, len(recorder.scheduled))
	for i, invocation := range recorder.scheduled {
		scheduledIDs[i] = invocation.ID
	}
	assert.ElementsMatch(t, scheduledIDs, recorder.finished)
	expectedOutput := map[string]string{
		"default": "parent",
		"first":   "first",
		"second":  "second",
		"third":   "third",
		"shared":  "shared",
	}
	for name, invocation := range byName {
		require.Contains(t, recorder.outputs, invocation.ID)
		assert.Contains(t, recorder.outputs[invocation.ID].String(), expectedOutput[name])
	}
}

func TestTaskLifecycleReportsFailfastCancellation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  default:
    failfast: true
    deps: [fail, slow]
  fail:
    cmds:
      - sleep 0.1
      - exit 1
  slow:
    cmds:
      - sleep 5
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o600))

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
		task.WithSilent(true),
		task.WithForce(true),
	)
	require.NoError(t, e.Setup())
	recorder := &lifecycleRecorder{}
	e.Output = recorder

	require.Error(t, e.Run(t.Context(), &task.Call{Task: "default"}))
	byName := make(map[string]output.TaskInvocation)
	for _, invocation := range recorder.started {
		byName[invocation.Name] = invocation
	}
	require.Contains(t, byName, "slow")
	assert.ErrorIs(t, recorder.finishErrors[byName["slow"].ID], context.Canceled)
}

type lifecycleRecorder struct {
	mutex        sync.Mutex
	scheduled    []output.TaskInvocation
	started      []output.TaskInvocation
	finished     []uint64
	outputs      map[uint64]*bytes.Buffer
	joined       map[uint64]uint64
	finishErrors map[uint64]error
}

func (*lifecycleRecorder) WrapWriter(_ io.Writer, _ io.Writer, _ string, _ *templater.Cache) (io.Writer, io.Writer, output.CloseFunc) {
	return io.Discard, io.Discard, func(error) error { return nil }
}

func (r *lifecycleRecorder) WrapWriterForTask(_ io.Writer, _ io.Writer, task output.TaskInvocation, _ *templater.Cache) (io.Writer, io.Writer, output.CloseFunc) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.outputs == nil {
		r.outputs = make(map[uint64]*bytes.Buffer)
	}
	buffer := r.outputs[task.ID]
	if buffer == nil {
		buffer = &bytes.Buffer{}
		r.outputs[task.ID] = buffer
	}
	return buffer, buffer, func(error) error { return nil }
}

func (r *lifecycleRecorder) TaskStarted(task output.TaskInvocation) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.started = append(r.started, task)
}

func (r *lifecycleRecorder) TaskScheduled(task output.TaskInvocation) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.scheduled = append(r.scheduled, task)
}

func (r *lifecycleRecorder) TaskFinished(id uint64, err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.finished = append(r.finished, id)
	if r.finishErrors == nil {
		r.finishErrors = make(map[uint64]error)
	}
	r.finishErrors[id] = err
}

func (r *lifecycleRecorder) TaskJoined(id, ownerID uint64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.joined == nil {
		r.joined = make(map[uint64]uint64)
	}
	r.joined[id] = ownerID
}

func countInvocations(invocations []output.TaskInvocation, name string) int {
	count := 0
	for _, invocation := range invocations {
		if invocation.Name == name {
			count++
		}
	}
	return count
}
