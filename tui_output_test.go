package task_test

import (
	"bytes"
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
  first: echo first
  second: echo second
  third: echo third
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
	require.Len(t, recorder.started, 4)
	byName := make(map[string]output.TaskInvocation)
	for _, invocation := range recorder.started {
		byName[invocation.Name] = invocation
	}
	root := byName["default"]
	assert.Zero(t, root.ParentID)
	for _, name := range []string{"first", "second", "third"} {
		assert.Equal(t, root.ID, byName[name].ParentID, name)
	}
	assert.ElementsMatch(t, []uint64{
		byName["default"].ID,
		byName["first"].ID,
		byName["second"].ID,
		byName["third"].ID,
	}, recorder.finished)
	expectedOutput := map[string]string{
		"default": "parent",
		"first":   "first",
		"second":  "second",
		"third":   "third",
	}
	for name, invocation := range byName {
		require.Contains(t, recorder.outputs, invocation.ID)
		assert.Contains(t, recorder.outputs[invocation.ID].String(), expectedOutput[name])
	}
}

type lifecycleRecorder struct {
	mutex    sync.Mutex
	started  []output.TaskInvocation
	finished []uint64
	outputs  map[uint64]*bytes.Buffer
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

func (r *lifecycleRecorder) TaskFinished(id uint64, _ error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.finished = append(r.finished, id)
}
