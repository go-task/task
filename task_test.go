package task_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/experiments"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

func init() {
	_ = os.Setenv("NO_COLOR", "1")
}

// SyncBuffer is a threadsafe buffer for testing.
// Some times replace stdout/stderr with a buffer to capture output.
// stdout and stderr are threadsafe, but a regular bytes.Buffer is not.
// Using this instead helps prevents race conditions with output.
type SyncBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (sb *SyncBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

// fileContentTest provides a basic reusable test-case for running a Taskfile
// and inspect generated files.
type fileContentTest struct {
	Dir        string
	Entrypoint string
	Target     string
	TrimSpace  bool
	Files      map[string]string
}

func (fct fileContentTest) name(file string) string {
	return fmt.Sprintf("target=%q,file=%q", fct.Target, file)
}

func (fct fileContentTest) Run(t *testing.T) {
	for f := range fct.Files {
		_ = os.Remove(filepathext.SmartJoin(fct.Dir, f))
	}
	e := &task.Executor{
		Dir: fct.Dir,
		TempDir: task.TempDir{
			Remote:      filepathext.SmartJoin(fct.Dir, ".task"),
			Fingerprint: filepathext.SmartJoin(fct.Dir, ".task"),
		},
		Entrypoint: fct.Entrypoint,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
	}

	require.NoError(t, e.Setup(), "e.Setup()")
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: fct.Target}), "e.Run(target)")
	for name, expectContent := range fct.Files {
		t.Run(fct.name(name), func(t *testing.T) {
			path := filepathext.SmartJoin(e.Dir, name)
			b, err := os.ReadFile(path)
			require.NoError(t, err, "Error reading file")
			s := string(b)
			if fct.TrimSpace {
				s = strings.TrimSpace(s)
			}
			assert.Equal(t, expectContent, s, "unexpected file content in %s", path)
		})
	}
}

func TestEmptyTask(t *testing.T) {
	e := &task.Executor{
		Dir:    "testdata/empty_task",
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	require.NoError(t, e.Setup(), "e.Setup()")
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "default"}))
}

func TestEmptyTaskfile(t *testing.T) {
	e := &task.Executor{
		Dir:    "testdata/empty_taskfile",
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	require.Error(t, e.Setup(), "e.Setup()")
}

func TestEnv(t *testing.T) {
	t.Setenv("QUX", "from_os")
	tt := fileContentTest{
		Dir:       "testdata/env",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"local.txt":         "GOOS='linux' GOARCH='amd64' CGO_ENABLED='0'\n",
			"global.txt":        "FOO='foo' BAR='overriden' BAZ='baz'\n",
			"multiple_type.txt": "FOO='1' BAR='true' BAZ='1.1'\n",
			"not-overriden.txt": "QUX='from_os'\n",
		},
	}
	tt.Run(t)
	t.Setenv("TASK_X_ENV_PRECEDENCE", "1")
	experiments.EnvPrecedence = experiments.New("ENV_PRECEDENCE")
	ttt := fileContentTest{
		Dir:       "testdata/env",
		Target:    "overriden",
		TrimSpace: false,
		Files: map[string]string{
			"overriden.txt": "QUX='from_taskfile'\n",
		},
	}
	ttt.Run(t)
}

func TestVars(t *testing.T) {
	tt := fileContentTest{
		Dir:    "testdata/vars",
		Target: "default",
		Files: map[string]string{
			"missing-var.txt":  "\n",
			"var-order.txt":    "ABCDEF\n",
			"dependent-sh.txt": "123456\n",
			"with-call.txt":    "Hi, ABC123!\n",
			"from-dot-env.txt": "From .env file\n",
		},
	}
	tt.Run(t)
}

func TestSpecialVars(t *testing.T) {
	const dir = "testdata/special_vars"
	const subdir = "testdata/special_vars/subdir"
	toAbs := func(rel string) string {
		abs, err := filepath.Abs(rel)
		assert.NoError(t, err)
		return abs
	}

	tests := []struct {
		target   string
		expected string
	}{
		// Root
		{target: "print-task", expected: "print-task"},
		{target: "print-root-dir", expected: toAbs(dir)},
		{target: "print-taskfile", expected: toAbs(dir) + "/Taskfile.yml"},
		{target: "print-taskfile-dir", expected: toAbs(dir)},
		{target: "print-task-version", expected: "unknown"},
		// Included
		{target: "included:print-task", expected: "included:print-task"},
		{target: "included:print-root-dir", expected: toAbs(dir)},
		{target: "included:print-taskfile", expected: toAbs(dir) + "/included/Taskfile.yml"},
		{target: "included:print-taskfile-dir", expected: toAbs(dir) + "/included"},
		{target: "included:print-task-version", expected: "unknown"},
	}

	for _, dir := range []string{dir, subdir} {
		for _, test := range tests {
			t.Run(test.target, func(t *testing.T) {
				var buff bytes.Buffer
				e := &task.Executor{
					Dir:    dir,
					Stdout: &buff,
					Stderr: &buff,
					Silent: true,
				}
				require.NoError(t, e.Setup())
				require.NoError(t, e.Run(context.Background(), &ast.Call{Task: test.target}))
				assert.Equal(t, test.expected+"\n", buff.String())
			})
		}
	}
}

func TestConcurrency(t *testing.T) {
	const (
		dir    = "testdata/concurrency"
		target = "default"
	)

	e := &task.Executor{
		Dir:         dir,
		Stdout:      io.Discard,
		Stderr:      io.Discard,
		Concurrency: 1,
	}
	require.NoError(t, e.Setup(), "e.Setup()")
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: target}), "e.Run(target)")
}

func TestParams(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/params",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"hello.txt":       "Hello\n",
			"world.txt":       "World\n",
			"exclamation.txt": "!\n",
			"dep1.txt":        "Dependence1\n",
			"dep2.txt":        "Dependence2\n",
			"spanish.txt":     "¡Holla mundo!\n",
			"spanish-dep.txt": "¡Holla dependencia!\n",
			"portuguese.txt":  "Olá, mundo!\n",
			"portuguese2.txt": "Olá, mundo!\n",
			"german.txt":      "Welt!\n",
		},
	}
	tt.Run(t)
}

func TestDeps(t *testing.T) {
	const dir = "testdata/deps"

	files := []string{
		"d1.txt",
		"d2.txt",
		"d3.txt",
		"d11.txt",
		"d12.txt",
		"d13.txt",
		"d21.txt",
		"d22.txt",
		"d23.txt",
		"d31.txt",
		"d32.txt",
		"d33.txt",
	}

	for _, f := range files {
		_ = os.Remove(filepathext.SmartJoin(dir, f))
	}

	e := &task.Executor{
		Dir:    dir,
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "default"}))

	for _, f := range files {
		f = filepathext.SmartJoin(dir, f)
		if _, err := os.Stat(f); err != nil {
			t.Errorf("File %s should exist", f)
		}
	}
}

func TestStatus(t *testing.T) {
	const dir = "testdata/status"

	files := []string{
		"foo.txt",
		"bar.txt",
		"baz.txt",
	}

	for _, f := range files {
		path := filepathext.SmartJoin(dir, f)
		_ = os.Remove(path)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("File should not exist: %v", err)
		}
	}

	var buff bytes.Buffer
	e := &task.Executor{
		Dir: dir,
		TempDir: task.TempDir{
			Remote:      filepathext.SmartJoin(dir, ".task"),
			Fingerprint: filepathext.SmartJoin(dir, ".task"),
		},
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}
	require.NoError(t, e.Setup())
	// gen-foo creates foo.txt, and will always fail it's status check.
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-foo"}))
	// gen-foo creates bar.txt, and will pass its status-check the 3. time it
	// is run. It creates bar.txt, but also lists it as its source. So, the checksum
	// for the file won't match before after the second run as we the file
	// only exists after the first run.
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-bar"}))
	// gen-silent-baz is marked as being silent, and should only produce output
	// if e.Verbose is set to true.
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-silent-baz"}))

	for _, f := range files {
		if _, err := os.Stat(filepathext.SmartJoin(dir, f)); err != nil {
			t.Errorf("File should exist: %v", err)
		}
	}

	// Run gen-bar a second time to produce a checksum file that matches bar.txt
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-bar"}))

	// Run gen-bar a third time, to make sure we've triggered the status check.
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-bar"}))

	// We're silent, so no output should have been produced.
	assert.Empty(t, buff.String())

	// Now, let's remove source file, and run the task again to to prepare
	// for the next test.
	err := os.Remove(filepathext.SmartJoin(dir, "bar.txt"))
	require.NoError(t, err)
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-bar"}))
	buff.Reset()

	// Global silence switched of, so we should see output unless the task itself
	// is silent.
	e.Silent = false

	// all: not up-to-date
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-foo"}))
	assert.Equal(t, "task: [gen-foo] touch foo.txt", strings.TrimSpace(buff.String()))
	buff.Reset()
	// status: not up-to-date
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-foo"}))
	assert.Equal(t, "task: [gen-foo] touch foo.txt", strings.TrimSpace(buff.String()))
	buff.Reset()

	// sources: not up-to-date
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-bar"}))
	assert.Equal(t, "task: [gen-bar] touch bar.txt", strings.TrimSpace(buff.String()))
	buff.Reset()
	// all: up-to-date
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-bar"}))
	assert.Equal(t, `task: Task "gen-bar" is up to date`, strings.TrimSpace(buff.String()))
	buff.Reset()

	// sources: not up-to-date, no output produced.
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-silent-baz"}))
	assert.Empty(t, buff.String())

	// up-to-date, no output produced
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-silent-baz"}))
	assert.Empty(t, buff.String())

	e.Verbose = true
	// up-to-date, output produced due to Verbose mode.
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "gen-silent-baz"}))
	assert.Equal(t, `task: Task "gen-silent-baz" is up to date`, strings.TrimSpace(buff.String()))
	buff.Reset()
}

func TestPrecondition(t *testing.T) {
	const dir = "testdata/precondition"

	var buff bytes.Buffer
	e := &task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}

	// A precondition that has been met
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "foo"}))
	if buff.String() != "" {
		t.Errorf("Got Output when none was expected: %s", buff.String())
	}

	// A precondition that was not met
	require.Error(t, e.Run(context.Background(), &ast.Call{Task: "impossible"}))

	if buff.String() != "task: 1 != 0 obviously!\n" {
		t.Errorf("Wrong output message: %s", buff.String())
	}
	buff.Reset()

	// Calling a task with a precondition in a dependency fails the task
	require.Error(t, e.Run(context.Background(), &ast.Call{Task: "depends_on_impossible"}))

	if buff.String() != "task: 1 != 0 obviously!\n" {
		t.Errorf("Wrong output message: %s", buff.String())
	}
	buff.Reset()

	// Calling a task with a precondition in a cmd fails the task
	require.Error(t, e.Run(context.Background(), &ast.Call{Task: "executes_failing_task_as_cmd"}))
	if buff.String() != "task: 1 != 0 obviously!\n" {
		t.Errorf("Wrong output message: %s", buff.String())
	}
	buff.Reset()
}

func TestGenerates(t *testing.T) {
	const dir = "testdata/generates"

	const (
		srcTask        = "sub/src.txt"
		relTask        = "rel.txt"
		absTask        = "abs.txt"
		fileWithSpaces = "my text file.txt"
	)

	srcFile := filepathext.SmartJoin(dir, srcTask)

	for _, task := range []string{srcTask, relTask, absTask, fileWithSpaces} {
		path := filepathext.SmartJoin(dir, task)
		_ = os.Remove(path)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("File should not exist: %v", err)
		}
	}

	buff := bytes.NewBuffer(nil)
	e := &task.Executor{
		Dir:    dir,
		Stdout: buff,
		Stderr: buff,
	}
	require.NoError(t, e.Setup())

	for _, theTask := range []string{relTask, absTask, fileWithSpaces} {
		destFile := filepathext.SmartJoin(dir, theTask)
		upToDate := fmt.Sprintf("task: Task \"%s\" is up to date\n", srcTask) +
			fmt.Sprintf("task: Task \"%s\" is up to date\n", theTask)

		// Run task for the first time.
		require.NoError(t, e.Run(context.Background(), &ast.Call{Task: theTask}))

		if _, err := os.Stat(srcFile); err != nil {
			t.Errorf("File should exist: %v", err)
		}
		if _, err := os.Stat(destFile); err != nil {
			t.Errorf("File should exist: %v", err)
		}
		// Ensure task was not incorrectly found to be up-to-date on first run.
		if buff.String() == upToDate {
			t.Errorf("Wrong output message: %s", buff.String())
		}
		buff.Reset()

		// Re-run task to ensure it's now found to be up-to-date.
		require.NoError(t, e.Run(context.Background(), &ast.Call{Task: theTask}))
		if buff.String() != upToDate {
			t.Errorf("Wrong output message: %s", buff.String())
		}
		buff.Reset()
	}
}

func TestStatusChecksum(t *testing.T) {
	const dir = "testdata/checksum"

	tests := []struct {
		files []string
		task  string
	}{
		{[]string{"generated.txt", ".task/checksum/build"}, "build"},
		{[]string{"generated.txt", ".task/checksum/build-with-status"}, "build-with-status"},
	}

	for _, test := range tests {
		t.Run(test.task, func(t *testing.T) {
			for _, f := range test.files {
				_ = os.Remove(filepathext.SmartJoin(dir, f))

				_, err := os.Stat(filepathext.SmartJoin(dir, f))
				require.Error(t, err)
			}

			var buff bytes.Buffer
			tempdir := task.TempDir{
				Remote:      filepathext.SmartJoin(dir, ".task"),
				Fingerprint: filepathext.SmartJoin(dir, ".task"),
			}
			e := task.Executor{
				Dir:     dir,
				TempDir: tempdir,
				Stdout:  &buff,
				Stderr:  &buff,
			}
			require.NoError(t, e.Setup())

			require.NoError(t, e.Run(context.Background(), &ast.Call{Task: test.task}))
			for _, f := range test.files {
				_, err := os.Stat(filepathext.SmartJoin(dir, f))
				require.NoError(t, err)
			}

			// Capture the modification time, so we can ensure the checksum file
			// is not regenerated when the hash hasn't changed.
			s, err := os.Stat(filepathext.SmartJoin(tempdir.Fingerprint, "checksum/"+test.task))
			require.NoError(t, err)
			time := s.ModTime()

			buff.Reset()
			require.NoError(t, e.Run(context.Background(), &ast.Call{Task: test.task}))
			assert.Equal(t, `task: Task "`+test.task+`" is up to date`+"\n", buff.String())

			s, err = os.Stat(filepathext.SmartJoin(tempdir.Fingerprint, "checksum/"+test.task))
			require.NoError(t, err)
			assert.Equal(t, time, s.ModTime())
		})
	}
}

func TestAlias(t *testing.T) {
	const dir = "testdata/alias"

	data, err := os.ReadFile(filepathext.SmartJoin(dir, "alias.txt"))
	require.NoError(t, err)

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "f"}))
	assert.Equal(t, string(data), buff.String())
}

func TestDuplicateAlias(t *testing.T) {
	const dir = "testdata/alias"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())
	require.Error(t, e.Run(context.Background(), &ast.Call{Task: "x"}))
	assert.Equal(t, "", buff.String())
}

func TestAliasSummary(t *testing.T) {
	const dir = "testdata/alias"

	data, err := os.ReadFile(filepathext.SmartJoin(dir, "alias-summary.txt"))
	require.NoError(t, err)

	var buff bytes.Buffer
	e := task.Executor{
		Dir:     dir,
		Summary: true,
		Stdout:  &buff,
		Stderr:  &buff,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "f"}))
	assert.Equal(t, string(data), buff.String())
}

func TestLabelUpToDate(t *testing.T) {
	const dir = "testdata/label_uptodate"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "foo"}))
	assert.Contains(t, buff.String(), "foobar")
}

func TestLabelSummary(t *testing.T) {
	const dir = "testdata/label_summary"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:     dir,
		Summary: true,
		Stdout:  &buff,
		Stderr:  &buff,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "foo"}))
	assert.Contains(t, buff.String(), "foobar")
}

func TestLabelInStatus(t *testing.T) {
	const dir = "testdata/label_status"

	e := task.Executor{
		Dir: dir,
	}
	require.NoError(t, e.Setup())
	err := e.Status(context.Background(), &ast.Call{Task: "foo"})
	assert.ErrorContains(t, err, "foobar")
}

func TestLabelWithVariableExpansion(t *testing.T) {
	const dir = "testdata/label_var"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "foo"}))
	assert.Contains(t, buff.String(), "foobaz")
}

func TestLabelInSummary(t *testing.T) {
	const dir = "testdata/label_summary"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "foo"}))
	assert.Contains(t, buff.String(), "foobar")
}

func TestPromptInSummary(t *testing.T) {
	const dir = "testdata/prompt"
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"test short approval", "y\n", false},
		{"test long approval", "yes\n", false},
		{"test uppercase approval", "Y\n", false},
		{"test stops task", "n\n", true},
		{"test junk value stops task", "foobar\n", true},
		{"test Enter stops task", "\n", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var inBuff bytes.Buffer
			var outBuff bytes.Buffer
			var errBuff bytes.Buffer

			inBuff.Write([]byte(test.input))

			e := task.Executor{
				Dir:        dir,
				Stdin:      &inBuff,
				Stdout:     &outBuff,
				Stderr:     &errBuff,
				AssumeTerm: true,
			}
			require.NoError(t, e.Setup())

			err := e.Run(context.Background(), &ast.Call{Task: "foo"})

			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPromptWithIndirectTask(t *testing.T) {
	const dir = "testdata/prompt"
	var inBuff bytes.Buffer
	var outBuff bytes.Buffer
	var errBuff bytes.Buffer

	inBuff.Write([]byte("y\n"))

	e := task.Executor{
		Dir:        dir,
		Stdin:      &inBuff,
		Stdout:     &outBuff,
		Stderr:     &errBuff,
		AssumeTerm: true,
	}
	require.NoError(t, e.Setup())

	err := e.Run(context.Background(), &ast.Call{Task: "bar"})
	assert.Contains(t, outBuff.String(), "show-prompt")
	require.NoError(t, err)
}

func TestPromptAssumeYes(t *testing.T) {
	const dir = "testdata/prompt"
	tests := []struct {
		name      string
		assumeYes bool
	}{
		{"--yes flag should skip prompt", true},
		{"task should raise errors.TaskCancelledError", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var inBuff bytes.Buffer
			var outBuff bytes.Buffer
			var errBuff bytes.Buffer

			// always cancel the prompt so we can require.Error
			inBuff.Write([]byte("\n"))

			e := task.Executor{
				Dir:       dir,
				Stdin:     &inBuff,
				Stdout:    &outBuff,
				Stderr:    &errBuff,
				AssumeYes: test.assumeYes,
			}
			require.NoError(t, e.Setup())

			err := e.Run(context.Background(), &ast.Call{Task: "foo"})

			if !test.assumeYes {
				require.Error(t, err)
				return
			}
		})
	}
}

func TestNoLabelInList(t *testing.T) {
	const dir = "testdata/label_list"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())
	if _, err := e.ListTasks(task.ListOptions{ListOnlyTasksWithDescriptions: true}); err != nil {
		t.Error(err)
	}
	assert.Contains(t, buff.String(), "foo")
}

// task -al case 1: listAll list all tasks
func TestListAllShowsNoDesc(t *testing.T) {
	const dir = "testdata/list_mixed_desc"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}

	require.NoError(t, e.Setup())

	var title string
	if _, err := e.ListTasks(task.ListOptions{ListAllTasks: true}); err != nil {
		t.Error(err)
	}
	for _, title = range []string{
		"foo",
		"voo",
		"doo",
	} {
		assert.Contains(t, buff.String(), title)
	}
}

// task -al case 2: !listAll list some tasks (only those with desc)
func TestListCanListDescOnly(t *testing.T) {
	const dir = "testdata/list_mixed_desc"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}

	require.NoError(t, e.Setup())
	if _, err := e.ListTasks(task.ListOptions{ListOnlyTasksWithDescriptions: true}); err != nil {
		t.Error(err)
	}

	var title string
	assert.Contains(t, buff.String(), "foo")
	for _, title = range []string{
		"voo",
		"doo",
	} {
		assert.NotContains(t, buff.String(), title)
	}
}

func TestListDescInterpolation(t *testing.T) {
	const dir = "testdata/list_desc_interpolation"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}

	require.NoError(t, e.Setup())
	if _, err := e.ListTasks(task.ListOptions{ListOnlyTasksWithDescriptions: true}); err != nil {
		t.Error(err)
	}

	assert.Contains(t, buff.String(), "foo-var")
	assert.Contains(t, buff.String(), "bar-var")
}

func TestStatusVariables(t *testing.T) {
	const dir = "testdata/status_vars"

	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))
	_ = os.Remove(filepathext.SmartJoin(dir, "generated.txt"))

	var buff bytes.Buffer
	e := task.Executor{
		Dir: dir,
		TempDir: task.TempDir{
			Remote:      filepathext.SmartJoin(dir, ".task"),
			Fingerprint: filepathext.SmartJoin(dir, ".task"),
		},
		Stdout:  &buff,
		Stderr:  &buff,
		Silent:  false,
		Verbose: true,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "build"}))

	assert.Contains(t, buff.String(), "3e464c4b03f4b65d740e1e130d4d108a")

	inf, err := os.Stat(filepathext.SmartJoin(dir, "source.txt"))
	require.NoError(t, err)
	ts := fmt.Sprintf("%d", inf.ModTime().Unix())
	tf := inf.ModTime().String()

	assert.Contains(t, buff.String(), ts)
	assert.Contains(t, buff.String(), tf)
}

func TestInit(t *testing.T) {
	const dir = "testdata/init"
	file := filepathext.SmartJoin(dir, "Taskfile.yml")

	_ = os.Remove(file)
	if _, err := os.Stat(file); err == nil {
		t.Errorf("Taskfile.yml should not exist")
	}

	if err := task.InitTaskfile(io.Discard, dir); err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Errorf("Taskfile.yml should exist")
	}
	_ = os.Remove(file)
}

func TestCyclicDep(t *testing.T) {
	const dir = "testdata/cyclic"

	e := task.Executor{
		Dir:    dir,
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	require.NoError(t, e.Setup())
	assert.IsType(t, &errors.TaskCalledTooManyTimesError{}, e.Run(context.Background(), &ast.Call{Task: "task-1"}))
}

func TestTaskVersion(t *testing.T) {
	tests := []struct {
		Dir     string
		Version *semver.Version
		wantErr bool
	}{
		{"testdata/version/v1", semver.MustParse("1"), true},
		{"testdata/version/v2", semver.MustParse("2"), true},
		{"testdata/version/v3", semver.MustParse("3"), false},
	}

	for _, test := range tests {
		t.Run(test.Dir, func(t *testing.T) {
			e := task.Executor{
				Dir:    test.Dir,
				Stdout: io.Discard,
				Stderr: io.Discard,
			}
			err := e.Setup()
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.Version, e.Taskfile.Version)
			assert.Equal(t, 2, e.Taskfile.Tasks.Len())
		})
	}
}

func TestTaskIgnoreErrors(t *testing.T) {
	const dir = "testdata/ignore_errors"

	e := task.Executor{
		Dir:    dir,
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	require.NoError(t, e.Setup())

	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "task-should-pass"}))
	require.Error(t, e.Run(context.Background(), &ast.Call{Task: "task-should-fail"}))
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "cmd-should-pass"}))
	require.Error(t, e.Run(context.Background(), &ast.Call{Task: "cmd-should-fail"}))
}

func TestExpand(t *testing.T) {
	const dir = "testdata/expand"

	home, err := os.UserHomeDir()
	if err != nil {
		t.Errorf("Couldn't get $HOME: %v", err)
	}
	var buff bytes.Buffer

	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "pwd"}))
	assert.Equal(t, home, strings.TrimSpace(buff.String()))
}

func TestDry(t *testing.T) {
	const dir = "testdata/dry"

	file := filepathext.SmartJoin(dir, "file.txt")
	_ = os.Remove(file)

	var buff bytes.Buffer

	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Dry:    true,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "build"}))

	assert.Equal(t, "task: [build] touch file.txt", strings.TrimSpace(buff.String()))
	if _, err := os.Stat(file); err == nil {
		t.Errorf("File should not exist %s", file)
	}
}

// TestDryChecksum tests if the checksum file is not being written to disk
// if the dry mode is enabled.
func TestDryChecksum(t *testing.T) {
	const dir = "testdata/dry_checksum"

	checksumFile := filepathext.SmartJoin(dir, ".task/checksum/default")
	_ = os.Remove(checksumFile)

	e := task.Executor{
		Dir: dir,
		TempDir: task.TempDir{
			Remote:      filepathext.SmartJoin(dir, ".task"),
			Fingerprint: filepathext.SmartJoin(dir, ".task"),
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
		Dry:    true,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "default"}))

	_, err := os.Stat(checksumFile)
	require.Error(t, err, "checksum file should not exist")

	e.Dry = false
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "default"}))
	_, err = os.Stat(checksumFile)
	require.NoError(t, err, "checksum file should exist")
}

func TestIncludes(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/includes",
		Target:    "default",
		TrimSpace: true,
		Files: map[string]string{
			"main.txt":                                  "main",
			"included_directory.txt":                    "included_directory",
			"included_directory_without_dir.txt":        "included_directory_without_dir",
			"included_taskfile_without_dir.txt":         "included_taskfile_without_dir",
			"./module2/included_directory_with_dir.txt": "included_directory_with_dir",
			"./module2/included_taskfile_with_dir.txt":  "included_taskfile_with_dir",
			"os_include.txt":                            "os",
		},
	}
	tt.Run(t)
}

func TestIncludesMultiLevel(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/includes_multi_level",
		Target:    "default",
		TrimSpace: true,
		Files: map[string]string{
			"called_one.txt":   "one",
			"called_two.txt":   "two",
			"called_three.txt": "three",
		},
	}
	tt.Run(t)
}

func TestIncludesRemote(t *testing.T) {
	dir := "testdata/includes_remote"

	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	tcs := []struct {
		firstRemote  string
		secondRemote string
	}{
		{
			firstRemote:  srv.URL + "/first/Taskfile.yml",
			secondRemote: srv.URL + "/first/second/Taskfile.yml",
		},
		{
			firstRemote:  srv.URL + "/first/Taskfile.yml",
			secondRemote: "./second/Taskfile.yml",
		},
	}

	tasks := []string{
		"first:write-file",
		"first:second:write-file",
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Setenv("FIRST_REMOTE_URL", tc.firstRemote)
			t.Setenv("SECOND_REMOTE_URL", tc.secondRemote)

			var buff SyncBuffer

			executors := []struct {
				name     string
				executor *task.Executor
			}{
				{
					name: "online, always download",
					executor: &task.Executor{
						Dir:      dir,
						Stdout:   &buff,
						Stderr:   &buff,
						Timeout:  time.Minute,
						Insecure: true,
						Logger:   &logger.Logger{Stdout: &buff, Stderr: &buff, Verbose: true},

						// Without caching
						AssumeYes: true,
						Download:  true,
					},
				},
				{
					name: "offline, use cache",
					executor: &task.Executor{
						Dir:      dir,
						Stdout:   &buff,
						Stderr:   &buff,
						Timeout:  time.Minute,
						Insecure: true,
						Logger:   &logger.Logger{Stdout: &buff, Stderr: &buff, Verbose: true},

						// With caching
						AssumeYes: false,
						Download:  false,
						Offline:   true,
					},
				},
			}

			for j, e := range executors {
				t.Run(fmt.Sprint(j), func(t *testing.T) {
					require.NoError(t, e.executor.Setup())

					for k, task := range tasks {
						t.Run(task, func(t *testing.T) {
							expectedContent := fmt.Sprint(random.Int63())
							t.Setenv("CONTENT", expectedContent)

							outputFile := fmt.Sprintf("%d.%d.txt", i, k)
							t.Setenv("OUTPUT_FILE", outputFile)

							path := filepath.Join(dir, outputFile)
							require.NoError(t, os.RemoveAll(path))

							require.NoError(t, e.executor.Run(context.Background(), &ast.Call{Task: task}))

							actualContent, err := os.ReadFile(path)
							require.NoError(t, err)
							assert.Equal(t, expectedContent, strings.TrimSpace(string(actualContent)))
						})
					}
				})
			}

			t.Log("\noutput:\n", buff.buf.String())
		})
	}
}

func TestIncludeCycle(t *testing.T) {
	const dir = "testdata/includes_cycle"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}

	err := e.Setup()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task: include cycle detected between")
}

func TestIncludesIncorrect(t *testing.T) {
	const dir = "testdata/includes_incorrect"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}

	err := e.Setup()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to parse testdata/includes_incorrect/incomplete.yml:", err.Error())
}

func TestIncludesEmptyMain(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/includes_empty",
		Target:    "included:default",
		TrimSpace: true,
		Files: map[string]string{
			"file.txt": "default",
		},
	}
	tt.Run(t)
}

func TestIncludesDependencies(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/includes_deps",
		Target:    "default",
		TrimSpace: true,
		Files: map[string]string{
			"default.txt":     "default",
			"called_dep.txt":  "called_dep",
			"called_task.txt": "called_task",
		},
	}
	tt.Run(t)
}

func TestIncludesCallingRoot(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/includes_call_root_task",
		Target:    "included:call-root",
		TrimSpace: true,
		Files: map[string]string{
			"root_task.txt": "root task",
		},
	}
	tt.Run(t)
}

func TestIncludesOptional(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/includes_optional",
		Target:    "default",
		TrimSpace: true,
		Files: map[string]string{
			"called_dep.txt": "called_dep",
		},
	}
	tt.Run(t)
}

func TestIncludesOptionalImplicitFalse(t *testing.T) {
	const dir = "testdata/includes_optional_implicit_false"
	wd, _ := os.Getwd()

	message := "stat %s/%s/TaskfileOptional.yml: no such file or directory"
	expected := fmt.Sprintf(message, wd, dir)

	e := task.Executor{
		Dir:    dir,
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	err := e.Setup()
	require.Error(t, err)
	assert.Equal(t, expected, err.Error())
}

func TestIncludesOptionalExplicitFalse(t *testing.T) {
	const dir = "testdata/includes_optional_explicit_false"
	wd, _ := os.Getwd()

	message := "stat %s/%s/TaskfileOptional.yml: no such file or directory"
	expected := fmt.Sprintf(message, wd, dir)

	e := task.Executor{
		Dir:    dir,
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	err := e.Setup()
	require.Error(t, err)
	assert.Equal(t, expected, err.Error())
}

func TestIncludesFromCustomTaskfile(t *testing.T) {
	tt := fileContentTest{
		Entrypoint: "testdata/includes_yaml/Custom.ext",
		Dir:        "testdata/includes_yaml",
		Target:     "default",
		TrimSpace:  true,
		Files: map[string]string{
			"main.txt":                         "main",
			"included_with_yaml_extension.txt": "included_with_yaml_extension",
			"included_with_custom_file.txt":    "included_with_custom_file",
		},
	}
	tt.Run(t)
}

func TestIncludesRelativePath(t *testing.T) {
	const dir = "testdata/includes_rel_path"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}

	require.NoError(t, e.Setup())

	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "common:pwd"}))
	assert.Contains(t, buff.String(), "testdata/includes_rel_path/common")

	buff.Reset()
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "included:common:pwd"}))
	assert.Contains(t, buff.String(), "testdata/includes_rel_path/common")
}

func TestIncludesInternal(t *testing.T) {
	const dir = "testdata/internal_task"
	tests := []struct {
		name           string
		task           string
		expectedErr    bool
		expectedOutput string
	}{
		{"included internal task via task", "task-1", false, "Hello, World!\n"},
		{"included internal task via dep", "task-2", false, "Hello, World!\n"},
		{"included internal direct", "included:task-3", true, "task: No tasks with description available. Try --list-all to list all tasks\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buff bytes.Buffer
			e := task.Executor{
				Dir:    dir,
				Stdout: &buff,
				Stderr: &buff,
				Silent: true,
			}
			require.NoError(t, e.Setup())

			err := e.Run(context.Background(), &ast.Call{Task: test.task})
			if test.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectedOutput, buff.String())
		})
	}
}

func TestIncludesInterpolation(t *testing.T) {
	const dir = "testdata/includes_interpolation"
	tests := []struct {
		name           string
		task           string
		expectedErr    bool
		expectedOutput string
	}{
		{"include", "include", false, "include\n"},
		{"include_with_env_variable", "include-with-env-variable", false, "include_with_env_variable\n"},
		{"include_with_dir", "include-with-dir", false, "included\n"},
	}
	t.Setenv("MODULE", "included")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buff bytes.Buffer
			e := task.Executor{
				Dir:    filepath.Join(dir, test.name),
				Stdout: &buff,
				Stderr: &buff,
				Silent: true,
			}
			require.NoError(t, e.Setup())

			err := e.Run(context.Background(), &ast.Call{Task: test.task})
			if test.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectedOutput, buff.String())
		})
	}
}

func TestIncludedTaskfileVarMerging(t *testing.T) {
	const dir = "testdata/included_taskfile_var_merging"
	tests := []struct {
		name           string
		task           string
		expectedOutput string
	}{
		{"foo", "foo:pwd", "included_taskfile_var_merging/foo\n"},
		{"bar", "bar:pwd", "included_taskfile_var_merging/bar\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buff bytes.Buffer
			e := task.Executor{
				Dir:    dir,
				Stdout: &buff,
				Stderr: &buff,
				Silent: true,
			}
			require.NoError(t, e.Setup())

			err := e.Run(context.Background(), &ast.Call{Task: test.task})
			require.NoError(t, err)
			assert.Contains(t, buff.String(), test.expectedOutput)
		})
	}
}

func TestInternalTask(t *testing.T) {
	const dir = "testdata/internal_task"
	tests := []struct {
		name           string
		task           string
		expectedErr    bool
		expectedOutput string
	}{
		{"internal task via task", "task-1", false, "Hello, World!\n"},
		{"internal task via dep", "task-2", false, "Hello, World!\n"},
		{"internal direct", "task-3", true, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buff bytes.Buffer
			e := task.Executor{
				Dir:    dir,
				Stdout: &buff,
				Stderr: &buff,
				Silent: true,
			}
			require.NoError(t, e.Setup())

			err := e.Run(context.Background(), &ast.Call{Task: test.task})
			if test.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expectedOutput, buff.String())
		})
	}
}

func TestIncludesShadowedDefault(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/includes_shadowed_default",
		Target:    "included",
		TrimSpace: true,
		Files: map[string]string{
			"file.txt": "shadowed",
		},
	}
	tt.Run(t)
}

func TestIncludesUnshadowedDefault(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/includes_unshadowed_default",
		Target:    "included",
		TrimSpace: true,
		Files: map[string]string{
			"file.txt": "included",
		},
	}
	tt.Run(t)
}

func TestSupportedFileNames(t *testing.T) {
	fileNames := []string{
		"Taskfile.yml",
		"Taskfile.yaml",
		"Taskfile.dist.yml",
		"Taskfile.dist.yaml",
	}
	for _, fileName := range fileNames {
		t.Run(fileName, func(t *testing.T) {
			tt := fileContentTest{
				Dir:       fmt.Sprintf("testdata/file_names/%s", fileName),
				Target:    "default",
				TrimSpace: true,
				Files: map[string]string{
					"output.txt": "hello",
				},
			}
			tt.Run(t)
		})
	}
}

func TestSummary(t *testing.T) {
	const dir = "testdata/summary"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:     dir,
		Stdout:  &buff,
		Stderr:  &buff,
		Summary: true,
		Silent:  true,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "task-with-summary"}, &ast.Call{Task: "other-task-with-summary"}))

	data, err := os.ReadFile(filepathext.SmartJoin(dir, "task-with-summary.txt"))
	require.NoError(t, err)

	expectedOutput := string(data)
	if runtime.GOOS == "windows" {
		expectedOutput = strings.ReplaceAll(expectedOutput, "\r\n", "\n")
	}

	assert.Equal(t, expectedOutput, buff.String())
}

func TestWhenNoDirAttributeItRunsInSameDirAsTaskfile(t *testing.T) {
	const expected = "dir"
	const dir = "testdata/" + expected
	var out bytes.Buffer
	e := &task.Executor{
		Dir:    dir,
		Stdout: &out,
		Stderr: &out,
	}

	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "whereami"}))

	// got should be the "dir" part of "testdata/dir"
	got := strings.TrimSuffix(filepath.Base(out.String()), "\n")
	assert.Equal(t, expected, got, "Mismatch in the working directory")
}

func TestWhenDirAttributeAndDirExistsItRunsInThatDir(t *testing.T) {
	const expected = "exists"
	const dir = "testdata/dir/explicit_exists"
	var out bytes.Buffer
	e := &task.Executor{
		Dir:    dir,
		Stdout: &out,
		Stderr: &out,
	}

	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "whereami"}))

	got := strings.TrimSuffix(filepath.Base(out.String()), "\n")
	assert.Equal(t, expected, got, "Mismatch in the working directory")
}

func TestWhenDirAttributeItCreatesMissingAndRunsInThatDir(t *testing.T) {
	const expected = "createme"
	const dir = "testdata/dir/explicit_doesnt_exist/"
	const toBeCreated = dir + expected
	const target = "whereami"
	var out bytes.Buffer
	e := &task.Executor{
		Dir:    dir,
		Stdout: &out,
		Stderr: &out,
	}

	// Ensure that the directory to be created doesn't actually exist.
	_ = os.RemoveAll(toBeCreated)
	if _, err := os.Stat(toBeCreated); err == nil {
		t.Errorf("Directory should not exist: %v", err)
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: target}))

	got := strings.TrimSuffix(filepath.Base(out.String()), "\n")
	assert.Equal(t, expected, got, "Mismatch in the working directory")

	// Clean-up after ourselves only if no error.
	_ = os.RemoveAll(toBeCreated)
}

func TestDynamicVariablesRunOnTheNewCreatedDir(t *testing.T) {
	const expected = "created"
	const dir = "testdata/dir/dynamic_var_on_created_dir/"
	const toBeCreated = dir + expected
	const target = "default"
	var out bytes.Buffer
	e := &task.Executor{
		Dir:    dir,
		Stdout: &out,
		Stderr: &out,
	}

	// Ensure that the directory to be created doesn't actually exist.
	_ = os.RemoveAll(toBeCreated)
	if _, err := os.Stat(toBeCreated); err == nil {
		t.Errorf("Directory should not exist: %v", err)
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: target}))

	got := strings.TrimSuffix(filepath.Base(out.String()), "\n")
	assert.Equal(t, expected, got, "Mismatch in the working directory")

	// Clean-up after ourselves only if no error.
	_ = os.RemoveAll(toBeCreated)
}

func TestDynamicVariablesShouldRunOnTheTaskDir(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/dir/dynamic_var",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"subdirectory/from_root_taskfile.txt":          "subdirectory\n",
			"subdirectory/from_included_taskfile.txt":      "subdirectory\n",
			"subdirectory/from_included_taskfile_task.txt": "subdirectory\n",
			"subdirectory/from_interpolated_dir.txt":       "subdirectory\n",
		},
	}
	tt.Run(t)
}

func TestDisplaysErrorOnVersion1Schema(t *testing.T) {
	e := task.Executor{
		Dir:    "testdata/version/v1",
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	err := e.Setup()
	require.Error(t, err)
	assert.Regexp(t, regexp.MustCompile(`task: Invalid schema version in Taskfile \".*testdata\/version\/v1\/Taskfile\.yml\":\nSchema version \(1\.0\.0\) no longer supported\. Please use v3 or above`), err.Error())
}

func TestDisplaysErrorOnVersion2Schema(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/version/v2",
		Stdout: io.Discard,
		Stderr: &buff,
	}
	err := e.Setup()
	require.Error(t, err)
	assert.Regexp(t, regexp.MustCompile(`task: Invalid schema version in Taskfile \".*testdata\/version\/v2\/Taskfile\.yml\":\nSchema version \(2\.0\.0\) no longer supported\. Please use v3 or above`), err.Error())
}

func TestShortTaskNotation(t *testing.T) {
	const dir = "testdata/short_task_notation"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "default"}))
	assert.Equal(t, "string-slice-1\nstring-slice-2\nstring\n", buff.String())
}

func TestDotenvShouldIncludeAllEnvFiles(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/dotenv/default",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"include.txt": "INCLUDE1='from_include1' INCLUDE2='from_include2'\n",
		},
	}
	tt.Run(t)
}

func TestDotenvShouldErrorWhenIncludingDependantDotenvs(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:     "testdata/dotenv/error_included_envs",
		Summary: true,
		Stdout:  &buff,
		Stderr:  &buff,
	}

	err := e.Setup()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "move the dotenv")
}

func TestDotenvShouldAllowMissingEnv(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/dotenv/missing_env",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"include.txt": "INCLUDE1='' INCLUDE2=''\n",
		},
	}
	tt.Run(t)
}

func TestDotenvHasLocalEnvInPath(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/dotenv/local_env_in_path",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"var.txt": "VAR='var_in_dot_env_1'\n",
		},
	}
	tt.Run(t)
}

func TestDotenvHasLocalVarInPath(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/dotenv/local_var_in_path",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"var.txt": "VAR='var_in_dot_env_3'\n",
		},
	}
	tt.Run(t)
}

func TestDotenvHasEnvVarInPath(t *testing.T) {
	os.Setenv("ENV_VAR", "testing")

	tt := fileContentTest{
		Dir:       "testdata/dotenv/env_var_in_path",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"var.txt": "VAR='var_in_dot_env_2'\n",
		},
	}
	tt.Run(t)
}

func TestTaskDotenv(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/dotenv_task/default",
		Target:    "dotenv",
		TrimSpace: true,
		Files: map[string]string{
			"dotenv.txt": "foo",
		},
	}
	tt.Run(t)
}

func TestTaskDotenvFail(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/dotenv_task/default",
		Target:    "no-dotenv",
		TrimSpace: true,
		Files: map[string]string{
			"no-dotenv.txt": "global",
		},
	}
	tt.Run(t)
}

func TestTaskDotenvOverriddenByEnv(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/dotenv_task/default",
		Target:    "dotenv-overridden-by-env",
		TrimSpace: true,
		Files: map[string]string{
			"dotenv-overridden-by-env.txt": "overridden",
		},
	}
	tt.Run(t)
}

func TestTaskDotenvWithVarName(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/dotenv_task/default",
		Target:    "dotenv-with-var-name",
		TrimSpace: true,
		Files: map[string]string{
			"dotenv-with-var-name.txt": "foo",
		},
	}
	tt.Run(t)
}

func TestExitImmediately(t *testing.T) {
	const dir = "testdata/exit_immediately"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}
	require.NoError(t, e.Setup())

	require.Error(t, e.Run(context.Background(), &ast.Call{Task: "default"}))
	assert.Contains(t, buff.String(), `"this_should_fail": executable file not found in $PATH`)
}

func TestRunOnlyRunsJobsHashOnce(t *testing.T) {
	tt := fileContentTest{
		Dir:    "testdata/run",
		Target: "generate-hash",
		Files: map[string]string{
			"hash.txt": "starting 1\n1\n2\n",
		},
	}
	tt.Run(t)
}

func TestRunOnceSharedDeps(t *testing.T) {
	const dir = "testdata/run_once_shared_deps"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:      dir,
		Stdout:   &buff,
		Stderr:   &buff,
		ForceAll: true,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "build"}))

	rx := regexp.MustCompile(`task: \[service-[a,b]:library:build\] echo "build library"`)
	matches := rx.FindAllStringSubmatch(buff.String(), -1)
	assert.Len(t, matches, 1)
	assert.Contains(t, buff.String(), `task: [service-a:build] echo "build a"`)
	assert.Contains(t, buff.String(), `task: [service-b:build] echo "build b"`)
}

func TestDeferredCmds(t *testing.T) {
	const dir = "testdata/deferred"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	expectedOutputOrder := strings.TrimSpace(`
task: [task-2] echo 'cmd ran'
cmd ran
task: [task-2] exit 1
task: [task-2] echo 'failing' && exit 2
failing
task: [task-2] echo 'echo ran'
echo ran
task: [task-1] echo 'task-1 ran successfully'
task-1 ran successfully
`)
	require.Error(t, e.Run(context.Background(), &ast.Call{Task: "task-2"}))
	assert.Contains(t, buff.String(), expectedOutputOrder)
}

func TestExitCodeZero(t *testing.T) {
	const dir = "testdata/exit_code"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "exit-zero"}))
	assert.Equal(t, "EXIT_CODE=", strings.TrimSpace(buff.String()))
}

func TestExitCodeOne(t *testing.T) {
	const dir = "testdata/exit_code"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	require.Error(t, e.Run(context.Background(), &ast.Call{Task: "exit-one"}))
	assert.Equal(t, "EXIT_CODE=1", strings.TrimSpace(buff.String()))
}

func TestIgnoreNilElements(t *testing.T) {
	tests := []struct {
		name string
		dir  string
	}{
		{"nil cmd", "testdata/ignore_nil_elements/cmds"},
		{"nil dep", "testdata/ignore_nil_elements/deps"},
		{"nil include", "testdata/ignore_nil_elements/includes"},
		{"nil precondition", "testdata/ignore_nil_elements/preconditions"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buff bytes.Buffer
			e := task.Executor{
				Dir:    test.dir,
				Stdout: &buff,
				Stderr: &buff,
				Silent: true,
			}
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "default"}))
			assert.Equal(t, "string-slice-1\n", buff.String())
		})
	}
}

func TestOutputGroup(t *testing.T) {
	const dir = "testdata/output_group"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	expectedOutputOrder := strings.TrimSpace(`
task: [hello] echo 'Hello!'
::group::hello
Hello!
::endgroup::
task: [bye] echo 'Bye!'
::group::bye
Bye!
::endgroup::
`)
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "bye"}))
	t.Log(buff.String())
	assert.Equal(t, strings.TrimSpace(buff.String()), expectedOutputOrder)
}

func TestOutputGroupErrorOnlySwallowsOutputOnSuccess(t *testing.T) {
	const dir = "testdata/output_group_error_only"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "passing"}))
	t.Log(buff.String())
	assert.Empty(t, buff.String())
}

func TestOutputGroupErrorOnlyShowsOutputOnFailure(t *testing.T) {
	const dir = "testdata/output_group_error_only"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	require.Error(t, e.Run(context.Background(), &ast.Call{Task: "failing"}))
	t.Log(buff.String())
	assert.Contains(t, "failing-output", strings.TrimSpace(buff.String()))
	assert.NotContains(t, "passing", strings.TrimSpace(buff.String()))
}

func TestIncludedVars(t *testing.T) {
	const dir = "testdata/include_with_vars"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	expectedOutputOrder := strings.TrimSpace(`
task: [included1:task1] echo "VAR_1 is included1-var1"
VAR_1 is included1-var1
task: [included1:task1] echo "VAR_2 is included-default-var2"
VAR_2 is included-default-var2
task: [included2:task1] echo "VAR_1 is included2-var1"
VAR_1 is included2-var1
task: [included2:task1] echo "VAR_2 is included-default-var2"
VAR_2 is included-default-var2
task: [included3:task1] echo "VAR_1 is included-default-var1"
VAR_1 is included-default-var1
task: [included3:task1] echo "VAR_2 is included-default-var2"
VAR_2 is included-default-var2
`)
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "task1"}))
	t.Log(buff.String())
	assert.Equal(t, strings.TrimSpace(buff.String()), expectedOutputOrder)
}

func TestIncludedVarsMultiLevel(t *testing.T) {
	const dir = "testdata/include_with_vars_multi_level"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	expectedOutputOrder := strings.TrimSpace(`
task: [lib:greet] echo 'Hello world'
Hello world
task: [foo:lib:greet] echo 'Hello foo'
Hello foo
task: [bar:lib:greet] echo 'Hello bar'
Hello bar
`)
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "default"}))
	t.Log(buff.String())
	assert.Equal(t, expectedOutputOrder, strings.TrimSpace(buff.String()))
}

func TestErrorCode(t *testing.T) {
	const dir = "testdata/error_code"
	tests := []struct {
		name     string
		task     string
		expected int
	}{
		{
			name:     "direct task",
			task:     "direct",
			expected: 42,
		}, {
			name:     "indirect task",
			task:     "indirect",
			expected: 42,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buff bytes.Buffer
			e := &task.Executor{
				Dir:    dir,
				Stdout: &buff,
				Stderr: &buff,
				Silent: true,
			}
			require.NoError(t, e.Setup())

			err := e.Run(context.Background(), &ast.Call{Task: test.task})
			require.Error(t, err)
			taskRunErr, ok := err.(*errors.TaskRunError)
			assert.True(t, ok, "cannot cast returned error to *task.TaskRunError")
			assert.Equal(t, test.expected, taskRunErr.TaskExitCode(), "unexpected exit code from task")
		})
	}
}

func TestEvaluateSymlinksInPaths(t *testing.T) {
	const dir = "testdata/evaluate_symlinks_in_paths"
	var buff bytes.Buffer
	e := &task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Silent: false,
	}
	tests := []struct {
		name     string
		task     string
		expected string
	}{
		{
			name:     "default (1)",
			task:     "default",
			expected: "task: [default] echo \"some job\"\nsome job",
		},
		{
			name:     "test-sym (1)",
			task:     "test-sym",
			expected: "task: [test-sym] echo \"shared file source changed\" > src/shared/b",
		},
		{
			name:     "default (2)",
			task:     "default",
			expected: "task: [default] echo \"some job\"\nsome job",
		},
		{
			name:     "default (3)",
			task:     "default",
			expected: `task: Task "default" is up to date`,
		},
		{
			name:     "reset",
			task:     "reset",
			expected: "task: [reset] echo \"shared file source\" > src/shared/b\ntask: [reset] echo \"file source\" > src/a",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, e.Setup())
			err := e.Run(context.Background(), &ast.Call{Task: test.task})
			require.NoError(t, err)
			assert.Equal(t, test.expected, strings.TrimSpace(buff.String()))
			buff.Reset()
		})
	}
	err := os.RemoveAll(dir + "/.task")
	require.NoError(t, err)
}

func TestTaskfileWalk(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		expected string
	}{
		{
			name:     "walk from root directory",
			dir:      "testdata/taskfile_walk",
			expected: "foo\n",
		}, {
			name:     "walk from sub directory",
			dir:      "testdata/taskfile_walk/foo",
			expected: "foo\n",
		}, {
			name:     "walk from sub sub directory",
			dir:      "testdata/taskfile_walk/foo/bar",
			expected: "foo\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buff bytes.Buffer
			e := task.Executor{
				Dir:    test.dir,
				Stdout: &buff,
				Stderr: &buff,
			}
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "default"}))
			assert.Equal(t, test.expected, buff.String())
		})
	}
}

func TestUserWorkingDirectory(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/user_working_dir",
		Stdout: &buff,
		Stderr: &buff,
	}
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "default"}))
	assert.Equal(t, fmt.Sprintf("%s\n", wd), buff.String())
}

func TestUserWorkingDirectoryWithIncluded(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	wd = filepathext.SmartJoin(wd, "testdata/user_working_dir_with_includes/somedir")

	var buff bytes.Buffer
	e := task.Executor{
		UserWorkingDir: wd,
		Dir:            "testdata/user_working_dir_with_includes",
		Stdout:         &buff,
		Stderr:         &buff,
	}

	require.NoError(t, err)
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "included:echo"}))
	assert.Equal(t, fmt.Sprintf("%s\n", wd), buff.String())
}

func TestPlatforms(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/platforms",
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())
	require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "build-" + runtime.GOOS}))
	assert.Equal(t, fmt.Sprintf("task: [build-%s] echo 'Running task on %s'\nRunning task on %s\n", runtime.GOOS, runtime.GOOS, runtime.GOOS), buff.String())
}

func TestPOSIXShellOptsGlobalLevel(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/shopts/global_level",
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	err := e.Run(context.Background(), &ast.Call{Task: "pipefail"})
	require.NoError(t, err)
	assert.Equal(t, "pipefail\ton\n", buff.String())
}

func TestPOSIXShellOptsTaskLevel(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/shopts/task_level",
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	err := e.Run(context.Background(), &ast.Call{Task: "pipefail"})
	require.NoError(t, err)
	assert.Equal(t, "pipefail\ton\n", buff.String())
}

func TestPOSIXShellOptsCommandLevel(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/shopts/command_level",
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	err := e.Run(context.Background(), &ast.Call{Task: "pipefail"})
	require.NoError(t, err)
	assert.Equal(t, "pipefail\ton\n", buff.String())
}

func TestBashShellOptsGlobalLevel(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/shopts/global_level",
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	err := e.Run(context.Background(), &ast.Call{Task: "globstar"})
	require.NoError(t, err)
	assert.Equal(t, "globstar\ton\n", buff.String())
}

func TestBashShellOptsTaskLevel(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/shopts/task_level",
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	err := e.Run(context.Background(), &ast.Call{Task: "globstar"})
	require.NoError(t, err)
	assert.Equal(t, "globstar\ton\n", buff.String())
}

func TestBashShellOptsCommandLevel(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/shopts/command_level",
		Stdout: &buff,
		Stderr: &buff,
	}
	require.NoError(t, e.Setup())

	err := e.Run(context.Background(), &ast.Call{Task: "globstar"})
	require.NoError(t, err)
	assert.Equal(t, "globstar\ton\n", buff.String())
}

func TestSplitArgs(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/split_args",
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}
	require.NoError(t, e.Setup())

	vars := &ast.Vars{}
	vars.Set("CLI_ARGS", ast.Var{Value: "foo bar 'foo bar baz'"})

	err := e.Run(context.Background(), &ast.Call{Task: "default", Vars: vars})
	require.NoError(t, err)
	assert.Equal(t, "3\n", buff.String())
}

func TestSingleCmdDep(t *testing.T) {
	tt := fileContentTest{
		Dir:    "testdata/single_cmd_dep",
		Target: "foo",
		Files: map[string]string{
			"foo.txt": "foo\n",
			"bar.txt": "bar\n",
		},
	}
	tt.Run(t)
}

func TestSilence(t *testing.T) {
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    "testdata/silent",
		Stdout: &buff,
		Stderr: &buff,
		Silent: false,
	}
	require.NoError(t, e.Setup())

	// First verify that the silent flag is in place.
	task, err := e.GetTask(&ast.Call{Task: "task-test-silent-calls-chatty-silenced"})
	require.NoError(t, err, "Unable to look up task task-test-silent-calls-chatty-silenced")
	require.True(t, task.Cmds[0].Silent, "The task task-test-silent-calls-chatty-silenced should have a silent call to chatty")

	// Then test the two basic cases where the task is silent or not.
	// A silenced task.
	err = e.Run(context.Background(), &ast.Call{Task: "silent"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "siWhile running lent: Expected not see output, because the task is silent")

	buff.Reset()

	// A chatty (not silent) task.
	err = e.Run(context.Background(), &ast.Call{Task: "chatty"})
	require.NoError(t, err)
	require.NotEmpty(t, buff.String(), "chWhile running atty: Expected to see output, because the task is not silent")

	buff.Reset()

	// Then test invoking the two task from other tasks.
	// A silenced task that calls a chatty task.
	err = e.Run(context.Background(), &ast.Call{Task: "task-test-silent-calls-chatty-non-silenced"})
	require.NoError(t, err)
	require.NotEmpty(t, buff.String(), "While running task-test-silent-calls-chatty-non-silenced: Expected to see output. The task is silenced, but the called task is not. Silence does not propagate to called tasks.")

	buff.Reset()

	// A silent task that does a silent call to a chatty task.
	err = e.Run(context.Background(), &ast.Call{Task: "task-test-silent-calls-chatty-silenced"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "While running task-test-silent-calls-chatty-silenced: Expected not to see output. The task calls chatty task, but the call is silenced.")

	buff.Reset()

	// A chatty task that does a call to a chatty task.
	err = e.Run(context.Background(), &ast.Call{Task: "task-test-chatty-calls-chatty-non-silenced"})
	require.NoError(t, err)
	require.NotEmpty(t, buff.String(), "While running task-test-chatty-calls-chatty-non-silenced: Expected to see output. Both caller and callee are chatty and not silenced.")

	buff.Reset()

	// A chatty task that does a silenced call to a chatty task.
	err = e.Run(context.Background(), &ast.Call{Task: "task-test-chatty-calls-chatty-silenced"})
	require.NoError(t, err)
	require.NotEmpty(t, buff.String(), "While running task-test-chatty-calls-chatty-silenced: Expected to see output. Call to a chatty task is silenced, but the parent task is not.")

	buff.Reset()

	// A chatty task with no cmd's of its own that does a silenced call to a chatty task.
	err = e.Run(context.Background(), &ast.Call{Task: "task-test-no-cmds-calls-chatty-silenced"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "While running task-test-no-cmds-calls-chatty-silenced: Expected not to see output. While the task itself is not silenced, it does not have any cmds and only does an invocation of a silenced task.")

	buff.Reset()

	// A chatty task that does a silenced invocation of a task.
	err = e.Run(context.Background(), &ast.Call{Task: "task-test-chatty-calls-silenced-cmd"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "While running task-test-chatty-calls-silenced-cmd: Expected not to see output. While the task itself is not silenced, its call to the chatty task is silent.")

	buff.Reset()

	// Then test calls via dependencies.
	// A silent task that depends on a chatty task.
	err = e.Run(context.Background(), &ast.Call{Task: "task-test-is-silent-depends-on-chatty-non-silenced"})
	require.NoError(t, err)
	require.NotEmpty(t, buff.String(), "While running task-test-is-silent-depends-on-chatty-non-silenced: Expected to see output. The task is silent and depends on a chatty task. Dependencies does not inherit silence.")

	buff.Reset()

	// A silent task that depends on a silenced chatty task.
	err = e.Run(context.Background(), &ast.Call{Task: "task-test-is-silent-depends-on-chatty-silenced"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "While running task-test-is-silent-depends-on-chatty-silenced: Expected not to see output. The task is silent and has a silenced dependency on a chatty task.")

	buff.Reset()

	// A chatty task that, depends on a silenced chatty task.
	err = e.Run(context.Background(), &ast.Call{Task: "task-test-is-chatty-depends-on-chatty-silenced"})
	require.NoError(t, err)
	require.Empty(t, buff.String(), "While running task-test-is-chatty-depends-on-chatty-silenced: Expected not to see output. The task is chatty but does not have commands and has a silenced dependency on a chatty task.")

	buff.Reset()
}

func TestForce(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		force    bool
		forceAll bool
	}{
		{
			name:  "force",
			force: true,
		},
		{
			name:     "force-all",
			forceAll: true,
		},
		{
			name:  "force with gentle force experiment",
			force: true,
			env: map[string]string{
				"TASK_X_GENTLE_FORCE": "1",
			},
		},
		{
			name:     "force-all with gentle force experiment",
			forceAll: true,
			env: map[string]string{
				"TASK_X_GENTLE_FORCE": "1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buff bytes.Buffer
			e := task.Executor{
				Dir:      "testdata/force",
				Stdout:   &buff,
				Stderr:   &buff,
				Force:    tt.force,
				ForceAll: tt.forceAll,
			}
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(context.Background(), &ast.Call{Task: "task-with-dep"}))
		})
	}
}

func TestForCmds(t *testing.T) {
	tests := []struct {
		name           string
		expectedOutput string
	}{
		{
			name:           "loop-explicit",
			expectedOutput: "a\nb\nc\n",
		},
		{
			name:           "loop-sources",
			expectedOutput: "bar\nfoo\n",
		},
		{
			name:           "loop-sources-glob",
			expectedOutput: "bar\nfoo\n",
		},
		{
			name:           "loop-vars",
			expectedOutput: "foo\nbar\n",
		},
		{
			name:           "loop-vars-sh",
			expectedOutput: "bar\nfoo\n",
		},
		{
			name:           "loop-task",
			expectedOutput: "foo\nbar\n",
		},
		{
			name:           "loop-task-as",
			expectedOutput: "foo\nbar\n",
		},
		{
			name:           "loop-different-tasks",
			expectedOutput: "1\n2\n3\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdOut bytes.Buffer
			var stdErr bytes.Buffer
			e := task.Executor{
				Dir:    "testdata/for/cmds",
				Stdout: &stdOut,
				Stderr: &stdErr,
				Silent: true,
				Force:  true,
			}
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(context.Background(), &ast.Call{Task: test.name}))
			assert.Equal(t, test.expectedOutput, stdOut.String())
		})
	}
}

func TestForDeps(t *testing.T) {
	tests := []struct {
		name                   string
		expectedOutputContains []string
	}{
		{
			name:                   "loop-explicit",
			expectedOutputContains: []string{"a\n", "b\n", "c\n"},
		},
		{
			name:                   "loop-sources",
			expectedOutputContains: []string{"bar\n", "foo\n"},
		},
		{
			name:                   "loop-sources-glob",
			expectedOutputContains: []string{"bar\n", "foo\n"},
		},
		{
			name:                   "loop-vars",
			expectedOutputContains: []string{"foo\n", "bar\n"},
		},
		{
			name:                   "loop-vars-sh",
			expectedOutputContains: []string{"bar\n", "foo\n"},
		},
		{
			name:                   "loop-task",
			expectedOutputContains: []string{"foo\n", "bar\n"},
		},
		{
			name:                   "loop-task-as",
			expectedOutputContains: []string{"foo\n", "bar\n"},
		},
		{
			name:                   "loop-different-tasks",
			expectedOutputContains: []string{"1\n", "2\n", "3\n"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// We need to use a sync buffer here as deps are run concurrently
			var buff SyncBuffer
			e := task.Executor{
				Dir:    "testdata/for/deps",
				Stdout: &buff,
				Stderr: &buff,
				Silent: true,
				Force:  true,
			}
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(context.Background(), &ast.Call{Task: test.name}))
			for _, expectedOutputContains := range test.expectedOutputContains {
				assert.Contains(t, buff.buf.String(), expectedOutputContains)
			}
		})
	}
}

func TestWildcard(t *testing.T) {
	tests := []struct {
		name           string
		call           string
		expectedOutput string
		wantErr        bool
	}{
		{
			name:           "basic wildcard",
			call:           "wildcard-foo",
			expectedOutput: "Hello foo\n",
		},
		{
			name:           "double wildcard",
			call:           "foo-wildcard-bar",
			expectedOutput: "Hello foo bar\n",
		},
		{
			name:           "store wildcard",
			call:           "start-foo",
			expectedOutput: "Starting foo\n",
		},
		{
			name:           "matches exactly",
			call:           "matches-exactly-*",
			expectedOutput: "I don't consume matches: []\n",
		},
		{
			name:    "no matches",
			call:    "no-match",
			wantErr: true,
		},
		{
			name:    "multiple matches",
			call:    "wildcard-foo-bar",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.call, func(t *testing.T) {
			var buff bytes.Buffer
			e := task.Executor{
				Dir:    "testdata/wildcards",
				Stdout: &buff,
				Stderr: &buff,
				Silent: true,
				Force:  true,
			}
			require.NoError(t, e.Setup())
			if test.wantErr {
				require.Error(t, e.Run(context.Background(), &ast.Call{Task: test.call}))
				return
			}
			require.NoError(t, e.Run(context.Background(), &ast.Call{Task: test.call}))
			assert.Equal(t, test.expectedOutput, buff.String())
		})
	}
}

func TestReference(t *testing.T) {
	tests := []struct {
		name           string
		call           string
		expectedOutput string
	}{
		{
			name:           "reference in command",
			call:           "ref-cmd",
			expectedOutput: "1\n",
		},
		{
			name:           "reference in dependency",
			call:           "ref-dep",
			expectedOutput: "1\n",
		},
		{
			name:           "reference using templating resolver",
			call:           "ref-resolver",
			expectedOutput: "1\n",
		},
		{
			name:           "reference using templating resolver and dynamic var",
			call:           "ref-resolver-sh",
			expectedOutput: "Alice has 3 children called Bob, Charlie, and Diane\n",
		},
	}

	for _, test := range tests {
		t.Run(test.call, func(t *testing.T) {
			var buff bytes.Buffer
			e := task.Executor{
				Dir:    "testdata/var_references",
				Stdout: &buff,
				Stderr: &buff,
				Silent: true,
				Force:  true,
			}
			require.NoError(t, e.Setup())
			require.NoError(t, e.Run(context.Background(), &ast.Call{Task: test.call}))
			assert.Equal(t, test.expectedOutput, buff.String())
		})
	}
}
