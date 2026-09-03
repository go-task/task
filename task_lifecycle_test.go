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
)

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
	e.Listener = recorder

	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "default"}))
	require.Len(t, recorder.scheduled, 6)
	require.Len(t, recorder.started, 5)
	byName := make(map[string]task.Invocation)
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
	e.Listener = recorder

	require.Error(t, e.Run(t.Context(), &task.Call{Task: "default"}))
	byName := make(map[string]task.Invocation)
	for _, invocation := range recorder.started {
		byName[invocation.Name] = invocation
	}
	require.Contains(t, byName, "slow")
	assert.ErrorIs(t, recorder.finishErrors[byName["slow"].ID], context.Canceled)
}

func TestTaskLifecycleSchedulesAllRequestedRootsBeforeExecution(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  build: echo build
  test: echo test
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
	e.Listener = recorder

	require.NoError(t, e.Run(
		t.Context(),
		&task.Call{Task: "build"},
		&task.Call{Task: "test"},
	))
	assert.Equal(t, 2, recorder.scheduledAtFirstStart)
	assert.Equal(t, []string{"build", "test"}, invocationNames(recorder.scheduled))
	assert.ElementsMatch(t, []uint64{recorder.scheduled[0].ID, recorder.scheduled[1].ID}, recorder.finished)
	for _, invocation := range recorder.scheduled {
		assert.Equal(t, invocation.ID, invocation.RootID)
		assert.Zero(t, invocation.ParentID)
	}
}

func TestTaskLifecycleFinishesRootThatFailsBeforeStarting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  lint:
    requires:
      vars: [FIX]
    cmds: [echo lint]
  typing: echo typing
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o600))

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
		task.WithSilent(true),
	)
	require.NoError(t, e.Setup())
	recorder := &lifecycleRecorder{}
	e.Listener = recorder

	err := e.Run(
		t.Context(),
		&task.Call{Task: "lint"},
		&task.Call{Task: "typing"},
	)
	require.Error(t, err)
	require.Len(t, recorder.scheduled, 2)
	require.Len(t, recorder.finished, 1)
	lint := recorder.scheduled[0]
	assert.Equal(t, lint.ID, recorder.finished[0])
	assert.Error(t, recorder.finishErrors[lint.ID])
}

type lifecycleRecorder struct {
	mutex        sync.Mutex
	scheduled    []task.Invocation
	started      []task.Invocation
	finished     []uint64
	outputs      map[uint64]*bytes.Buffer
	joined       map[uint64]uint64
	finishErrors map[uint64]error

	scheduledAtFirstStart int
}

func (*lifecycleRecorder) OwnsTerminal() bool { return false }

func (r *lifecycleRecorder) WriterFor(invocation task.Invocation) (io.Writer, io.Writer) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.outputs == nil {
		r.outputs = make(map[uint64]*bytes.Buffer)
	}
	buffer := r.outputs[invocation.ID]
	if buffer == nil {
		buffer = &bytes.Buffer{}
		r.outputs[invocation.ID] = buffer
	}
	return buffer, buffer
}

func (r *lifecycleRecorder) TaskStarted(invocation task.Invocation) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if len(r.started) == 0 {
		r.scheduledAtFirstStart = len(r.scheduled)
	}
	r.started = append(r.started, invocation)
}

func (r *lifecycleRecorder) TaskScheduled(invocation task.Invocation) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.scheduled = append(r.scheduled, invocation)
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

func countInvocations(invocations []task.Invocation, name string) int {
	count := 0
	for _, invocation := range invocations {
		if invocation.Name == name {
			count++
		}
	}
	return count
}

func invocationNames(invocations []task.Invocation) []string {
	names := make([]string, len(invocations))
	for i, invocation := range invocations {
		names[i] = invocation.Name
	}
	return names
}
