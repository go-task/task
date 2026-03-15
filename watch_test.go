//go:build watch
// +build watch

package task_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/filepathext"
)

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

// Create, Remove, Rename, Write
// In sources, not in sources
// sources is a ./**/*.txt

func TestWatchSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		action        string
		path          string
		expectRestart bool
	}{
		// Entry condition: file fubar/foo.txt exists.
		{"create", "fubar/bar.txt", true},
		{"remove", "fubar/foo.txt", true},
		{"rename", "fubar/foo.txt", true},
		{"write", "fubar/foo.txt", true},
		{"create", "fubar/bar.text", false},
		{"remove", "fubar/foo.text", false},
		{"rename", "fubar/foo.text", false},
		{"write", "fubar/foo.text", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("%s-%s", tc.action, tc.path), func(t *testing.T) {
			t.Parallel()

			checks := []string{`Started watching for tasks: default`, `echo "Task running!"`}

			// Setup the watch dir.
			tmpDir := t.TempDir()
			data, _ := os.ReadFile("testdata/watch/sources/Taskfile.yaml")
			os.WriteFile(filepath.Join(tmpDir, "Taskfile.yaml"), data, 0644)
			testFile := filepath.Join(tmpDir, "fubar/foo.txt")
			os.MkdirAll(filepath.Dir(testFile), 0755)
			os.WriteFile(testFile, []byte("hello world"), 0644)

			// Correct test case paths.
			tc.path = filepath.Join(tmpDir, tc.path)

			// Start the Task.
			var buf bytes.Buffer
			e := task.NewExecutor(
				task.WithDir(tmpDir),
				task.WithStdout(&buf),
				task.WithStderr(&buf),
				task.WithWatch(true),
				task.WithVerbose(true),
			)
			require.NoError(t, e.Setup())
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

			// Introduce the test condition.
			time.Sleep(200 * time.Millisecond)
			switch tc.action {
			case "create":
				f, _ := os.OpenFile(tc.path, os.O_CREATE|os.O_WRONLY, 0644)
				defer f.Close()
				f.WriteString("watch test")
				checks = append(checks, `watch event: CREATE`)

			case "remove":
				if !tc.expectRestart {
					f, _ := os.OpenFile(tc.path, os.O_CREATE|os.O_WRONLY, 0644)
					f.Close()
					time.Sleep(100 * time.Millisecond)
					checks = append(checks, `watch event: CREATE`)
				}
				os.Remove(tc.path)
				checks = append(checks, `watch event: REMOVE`)

			case "rename":
				if !tc.expectRestart {
					f, _ := os.OpenFile(tc.path, os.O_CREATE|os.O_WRONLY, 0644)
					f.Close()
					time.Sleep(100 * time.Millisecond)
					checks = append(checks, `watch event: CREATE`)
				}
				dir := filepath.Dir(tc.path)
				base := filepath.Base(tc.path)
				ext := filepath.Ext(base)
				name := base[:len(base)-len(ext)]
				_b := []byte(name)
				slices.Reverse(_b)
				name = string(_b)
				os.Rename(tc.path, filepath.Join(dir, name+ext))
				checks = append(checks, `watch event: RENAME`)

			case "write":
				f, _ := os.OpenFile(tc.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				defer f.Close()
				f.WriteString("watch test")
				checks = append(checks, `watch event: WRITE`)
			}

			// Observe the expected conditions.
			time.Sleep(200 * time.Millisecond)
			cancel()
			if tc.expectRestart {
				checks = append(checks, `echo "Task running!"`)
			} else {
				checks = append(checks, `skipped for file not in sources:`)
			}

			output := buf.String()
			t.Log(output)
			for _, check := range checks {
				if idx := strings.Index(output, check); idx == -1 {
					t.Log(output)
					t.Log(checks)
					t.Fatalf("Expected output not observed in sequence: %s", check)
				} else {
					output = output[idx+len(check):]
				}
			}
		})
	}
}
