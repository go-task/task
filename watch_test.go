//go:build watch

package task_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/filepathext"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestFileWatch(t *testing.T) {
	t.Parallel()

	const dir = "testdata/watch"
	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))
	_ = os.RemoveAll(filepathext.SmartJoin(dir, "src"))

	expectedOutput := strings.TrimSpace(`
task: Started watching for tasks: default
task: [default] echo "Task running!"
Task running!
task: task "default" finished running
task: [default] echo "Task running!"
Task running!
task: task "default" finished running
	`)

	var buff bytes.Buffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithWatch(true),
	)

	require.NoError(t, e.Setup())
	buff.Reset()

	dirPath := filepathext.SmartJoin(dir, "src")
	filePath := filepathext.SmartJoin(dirPath, "a")

	err := os.MkdirAll(dirPath, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filePath, []byte("test"), 0o644)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := e.Run(ctx, &task.Call{Task: "default"})
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)
	err = os.WriteFile(filePath, []byte("test updated"), 0o644)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	cancel()
	assert.Equal(t, expectedOutput, strings.TrimSpace(buff.String()))
}

// TestFileWatchNewDir checks that a directory created while watching is
// watched without having to wait for the periodic sources rescan.
func TestFileWatchNewDir(t *testing.T) {
	t.Parallel()

	const dir = "testdata/watch_newdir"
	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))
	_ = os.RemoveAll(filepathext.SmartJoin(dir, "src"))

	var buff syncBuffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithWatch(true),
	)
	require.NoError(t, e.Setup())

	dirPath := filepathext.SmartJoin(dir, "src")
	require.NoError(t, os.MkdirAll(dirPath, 0o755))
	require.NoError(t, os.WriteFile(filepathext.SmartJoin(dirPath, "a"), []byte("test"), 0o644))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := e.Run(ctx, &task.Call{Task: "default"}); err != nil {
			panic(err)
		}
	}()

	time.Sleep(300 * time.Millisecond)

	newDirPath := filepathext.SmartJoin(dirPath, "sub")
	require.NoError(t, os.MkdirAll(newDirPath, 0o755))
	require.NoError(t, os.WriteFile(filepathext.SmartJoin(newDirPath, "b"), []byte("test"), 0o644))

	require.Eventually(t, func() bool {
		return strings.Count(buff.String(), `"default" finished running`) >= 2
	}, 2*time.Second, 50*time.Millisecond)

	cancel()
}

// TestFileWatchSources checks that changes to files that are not task sources
// do not trigger a run.
func TestFileWatchSources(t *testing.T) {
	t.Parallel()

	const dir = "testdata/watch_sources"
	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))
	_ = os.RemoveAll(filepathext.SmartJoin(dir, "src"))

	expectedOutput := strings.TrimSpace(`
task: Started watching for tasks: default
task: [default] echo "Task running!"
Task running!
task: task "default" finished running
	`)

	var buff syncBuffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithWatch(true),
	)
	require.NoError(t, e.Setup())

	dirPath := filepathext.SmartJoin(dir, "src")
	require.NoError(t, os.MkdirAll(dirPath, 0o755))
	require.NoError(t, os.WriteFile(filepathext.SmartJoin(dirPath, "a.txt"), []byte("test"), 0o644))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := e.Run(ctx, &task.Call{Task: "default"}); err != nil {
			panic(err)
		}
	}()

	time.Sleep(300 * time.Millisecond)

	require.NoError(t, os.WriteFile(filepathext.SmartJoin(dirPath, "notes.md"), []byte("other"), 0o644))

	time.Sleep(500 * time.Millisecond)
	cancel()
	assert.Equal(t, expectedOutput, strings.TrimSpace(buff.String()))
}

// TestFileWatchRemove checks that removing a source file triggers a run.
func TestFileWatchRemove(t *testing.T) {
	t.Parallel()

	const dir = "testdata/watch_remove"
	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))
	_ = os.RemoveAll(filepathext.SmartJoin(dir, "src"))

	expectedOutput := strings.TrimSpace(`
task: Started watching for tasks: default
task: [default] echo "Task running!"
Task running!
task: task "default" finished running
task: [default] echo "Task running!"
Task running!
task: task "default" finished running
	`)

	var buff syncBuffer
	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithStdout(&buff),
		task.WithStderr(&buff),
		task.WithWatch(true),
	)
	require.NoError(t, e.Setup())

	dirPath := filepathext.SmartJoin(dir, "src")
	filePath := filepathext.SmartJoin(dirPath, "a")
	require.NoError(t, os.MkdirAll(dirPath, 0o755))
	require.NoError(t, os.WriteFile(filePath, []byte("test"), 0o644))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := e.Run(ctx, &task.Call{Task: "default"}); err != nil {
			panic(err)
		}
	}()

	time.Sleep(300 * time.Millisecond)

	require.NoError(t, os.Remove(filePath))

	time.Sleep(500 * time.Millisecond)
	cancel()
	assert.Equal(t, expectedOutput, strings.TrimSpace(buff.String()))
}

func TestShouldIgnore(t *testing.T) {
	t.Parallel()

	tt := []struct {
		path   string
		expect bool
	}{
		{"/.git/hooks", true},
		{"/.github/workflows/build.yaml", false},
	}

	for k, ct := range tt {
		ct := ct
		t.Run(fmt.Sprintf("ignore - %d", k), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, task.ShouldIgnore(ct.path), ct.expect)
		})
	}
}
