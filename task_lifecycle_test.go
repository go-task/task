package task_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

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
	e.Listener = recorder.listener()

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
	e.Listener = recorder.listener()

	require.Error(t, e.Run(t.Context(), &task.Call{Task: "default"}))
	byName := make(map[string]task.Invocation)
	for _, invocation := range recorder.started {
		byName[invocation.Name] = invocation
	}
	require.Contains(t, byName, "slow")
	assert.Equal(t, task.ResultCanceled, recorder.finishResults[byName["slow"].ID])
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
	e.Listener = recorder.listener()

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
	e.Listener = recorder.listener()

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
	assert.Equal(t, task.ResultFailed, recorder.finishResults[lint.ID])
}

type lifecycleRecorder struct {
	mutex           sync.Mutex
	scheduled       []task.Invocation
	started         []task.Invocation
	finished        []uint64
	outputs         map[uint64]*bytes.Buffer
	joined          map[uint64]uint64
	finishErrors    map[uint64]error
	finishResults   map[uint64]task.Result
	finishDurations map[uint64]time.Duration

	scheduledAtFirstStart int
}

// listener is the recorder as an executor listener. Building it from closures
// is what a client does, so the tests exercise the same shape as real callers.
func (r *lifecycleRecorder) listener() *task.Listener {
	return &task.Listener{
		Scheduled: func(invocation task.Invocation) {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			r.scheduled = append(r.scheduled, invocation)
		},
		Started: func(started task.Started) {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			if len(r.started) == 0 {
				r.scheduledAtFirstStart = len(r.scheduled)
			}
			r.started = append(r.started, started.Invocation)
		},
		Finished: func(finished task.Finished) {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			r.finished = append(r.finished, finished.ID)
			if r.finishErrors == nil {
				r.finishErrors = make(map[uint64]error)
				r.finishResults = make(map[uint64]task.Result)
				r.finishDurations = make(map[uint64]time.Duration)
			}
			r.finishErrors[finished.ID] = finished.Err
			r.finishResults[finished.ID] = finished.Result
			r.finishDurations[finished.ID] = finished.Duration
		},
		Joined: func(joined task.Joined) {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			if r.joined == nil {
				r.joined = make(map[uint64]uint64)
			}
			r.joined[joined.ID] = joined.OwnerID
		},
		OutputFor: func(invocation task.Invocation) (io.Writer, io.Writer) {
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
		},
	}
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

func TestResetRunStateLetsRunOnceTasksExecuteAgain(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  build:
    run: once
    cmds: [echo built]
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
	e.Listener = recorder.listener()

	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build"}))

	// Reusing the Executor without resetting joins the finished execution, so
	// the second run produces no output of its own.
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build"}))
	require.Len(t, recorder.joined, 1)

	e.ResetRunState()
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build"}))
	require.Len(t, recorder.joined, 1, "reset run should execute rather than join")

	// All three calls were scheduled, but the joined one never started or
	// produced output of its own: it adopted the first run's result.
	require.Len(t, recorder.scheduled, 3)
	require.Len(t, recorder.started, 2)
	for _, invocation := range recorder.started {
		buffer, ok := recorder.outputs[invocation.ID]
		require.True(t, ok, "started call %d produced no output", invocation.ID)
		assert.Contains(t, buffer.String(), "built")
	}
}

func TestTaskLifecycleReportsCallsThatCannotBeResolved(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  build:
    deps: [compile, typoo]
    cmds: [echo building]
  compile:
    cmds: [echo compiling]
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
	e.Listener = recorder.listener()

	require.Error(t, e.Run(t.Context(), &task.Call{Task: "build"}))

	// The dep never compiles, but it is still announced under the name written
	// in the Taskfile and its own failure is reported against it.
	byName := make(map[string]task.Invocation)
	for _, invocation := range recorder.scheduled {
		byName[invocation.Name] = invocation
	}
	require.Contains(t, byName, "typoo")
	err := recorder.finishErrors[byName["typoo"].ID]
	require.Error(t, err)
	assert.Equal(t, task.ResultFailed, recorder.finishResults[byName["typoo"].ID])
	assert.Contains(t, err.Error(), `Task "typoo" does not exist`)
}

func TestTaskLifecycleReportsSkippedCalls(t *testing.T) {
	t.Parallel()

	otherPlatform := "windows"
	if runtime.GOOS == "windows" {
		otherPlatform = "linux"
	}
	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  build:
    deps: [other-platform, condition-not-met, always]
  other-platform:
    platforms: [` + otherPlatform + `]
    cmds: [echo nope]
  condition-not-met:
    if: 'false'
    cmds: [echo nope]
  always:
    cmds: [echo yes]
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
	e.Listener = recorder.listener()

	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build"}))

	byName := make(map[string]task.Invocation)
	for _, invocation := range recorder.scheduled {
		byName[invocation.Name] = invocation
	}
	for _, name := range []string{"other-platform", "condition-not-met"} {
		require.Contains(t, byName, name)
		assert.Equal(t, task.ResultSkipped, recorder.finishResults[byName[name].ID], name)
		assert.NoError(t, recorder.finishErrors[byName[name].ID], "skipping is not a failure")
	}
	require.Contains(t, byName, "always")
	assert.Equal(t, task.ResultSucceeded, recorder.finishResults[byName["always"].ID])
}

func TestTaskLifecycleIdentifiesTasksByTheirTaskfileName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  build:
    label: Build the docs
    cmds: [echo building]
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
	e.Listener = recorder.listener()

	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build"}))

	require.Len(t, recorder.started, 1)
	started := recorder.started[0]
	// Name is for display and becomes the label. Task stays the key a client
	// needs to look the task back up.
	assert.Equal(t, "Build the docs", started.Name)
	assert.Equal(t, "build", started.Task)

	found, err := e.GetTask(&task.Call{Task: started.Task})
	require.NoError(t, err)
	assert.Equal(t, "Build the docs", found.Name())
}

func TestTaskLifecycleTimesTasksItself(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  slow:
    cmds: [sleep 0.2]
  never-runs:
    platforms: [plan9]
    cmds: [echo nope]
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
	e.Listener = recorder.listener()

	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "slow"}, &task.Call{Task: "never-runs"}))

	byName := make(map[string]task.Invocation)
	for _, invocation := range recorder.scheduled {
		byName[invocation.Task] = invocation
	}

	// The duration comes from the executor, so a client need not time the
	// events reaching it.
	assert.GreaterOrEqual(t, recorder.finishDurations[byName["slow"].ID], 200*time.Millisecond)
	// A call that never started has no duration to report.
	assert.Zero(t, recorder.finishDurations[byName["never-runs"].ID])
}

func TestListenerNeedsNoFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"),
		[]byte("version: '3'\ntasks:\n  build: echo built\n"), 0o600))

	var out bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&out),
		task.WithStderr(&out),
		task.WithSilent(true),
		task.WithForce(true),
	)
	require.NoError(t, e.Setup())

	// A listener that sets nothing observes nothing and changes nothing: output
	// still goes to the Executor's own streams.
	e.Listener = &task.Listener{}
	require.NoError(t, e.Run(t.Context(), &task.Call{Task: "build"}))
	assert.Contains(t, out.String(), "built")
}

func TestListenerLendsTheTerminalForPrompts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  deploy:
    requires:
      vars: [TARGET]
    cmds: [echo deploying]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o600))

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
		task.WithSilent(true),
		task.WithForce(true),
		task.WithInteractive(true),
		task.WithAssumeTerm(true),
	)
	require.NoError(t, e.Setup())

	// Declining tells us Task asked before prompting, without starting the
	// prompter, which needs a real terminal to drive.
	declined := errors.New("the client kept the terminal")
	lent := 0
	e.Listener = &task.Listener{
		OwnsScreen: true,
		RunInTerminal: func(func() error) error {
			lent++
			return declined
		},
	}

	err := e.Run(t.Context(), &task.Call{Task: "deploy"})
	require.ErrorIs(t, err, declined, "Task must honour a client that will not lend the terminal")
	assert.Equal(t, 1, lent, "prompting must borrow the terminal from the client")
}

func TestListenerRefusesPromptsItCannotLendTheTerminalFor(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	taskfile := `version: '3'
tasks:
  deploy:
    requires:
      vars: [TARGET]
    cmds: [echo deploying]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o600))

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
		task.WithSilent(true),
		task.WithForce(true),
		task.WithInteractive(true),
		task.WithAssumeTerm(true),
	)
	require.NoError(t, e.Setup())

	// A client drawing in a window has no terminal to lend, so it leaves
	// RunInTerminal nil and Task says so rather than prompting into the void.
	e.Listener = &task.Listener{OwnsScreen: true}

	err := e.Run(t.Context(), &task.Call{Task: "deploy"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs the terminal")
}
