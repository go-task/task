package task_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/taskfile"
)

func init() {
	_ = os.Setenv("NO_COLOR", "1")
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
		Dir:        fct.Dir,
		TempDir:    filepathext.SmartJoin(fct.Dir, ".task"),
		Entrypoint: fct.Entrypoint,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
	}
	assert.NoError(t, e.Setup(), "e.Setup()")
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: fct.Target}), "e.Run(target)")

	for name, expectContent := range fct.Files {
		t.Run(fct.name(name), func(t *testing.T) {
			path := filepathext.SmartJoin(fct.Dir, name)
			b, err := os.ReadFile(path)
			assert.NoError(t, err, "Error reading file")
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
	assert.NoError(t, e.Setup(), "e.Setup()")
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "default"}))
}

func TestEnv(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/env",
		Target:    "default",
		TrimSpace: false,
		Files: map[string]string{
			"local.txt":  "GOOS='linux' GOARCH='amd64' CGO_ENABLED='0'\n",
			"global.txt": "FOO='foo' BAR='overriden' BAZ='baz'\n",
		},
	}
	tt.Run(t)
}

func TestVarsV2(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/vars/v2",
		Target:    "default",
		TrimSpace: true,
		Files: map[string]string{
			"foo.txt":              "foo",
			"bar.txt":              "bar",
			"baz.txt":              "baz",
			"tmpl_foo.txt":         "foo",
			"tmpl_bar.txt":         "bar",
			"tmpl_foo2.txt":        "foo2",
			"tmpl_bar2.txt":        "bar2",
			"shtmpl_foo.txt":       "foo",
			"shtmpl_foo2.txt":      "foo2",
			"nestedtmpl_foo.txt":   "<no value>",
			"nestedtmpl_foo2.txt":  "foo2",
			"foo2.txt":             "foo2",
			"bar2.txt":             "bar2",
			"baz2.txt":             "baz2",
			"tmpl2_foo.txt":        "<no value>",
			"tmpl2_foo2.txt":       "foo2",
			"tmpl2_bar.txt":        "<no value>",
			"tmpl2_bar2.txt":       "bar2",
			"shtmpl2_foo.txt":      "<no value>",
			"shtmpl2_foo2.txt":     "foo2",
			"nestedtmpl2_foo2.txt": "<no value>",
			"override.txt":         "bar",
			"nested.txt":           "Taskvars-TaskfileVars-TaskVars",
			"task_name.txt":        "hello",
		},
	}
	tt.Run(t)
	// Ensure identical results when running hello task directly.
	tt.Target = "hello"
	tt.Run(t)
}

func TestVarsV3(t *testing.T) {
	tt := fileContentTest{
		Dir:    "testdata/vars/v3",
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

func TestMultilineVars(t *testing.T) {
	for _, dir := range []string{"testdata/vars/v2/multiline"} {
		tt := fileContentTest{
			Dir:       dir,
			Target:    "default",
			TrimSpace: false,
			Files: map[string]string{
				// Note:
				// - task does not strip a trailing newline from var entries
				// - task strips one trailing newline from shell output
				// - the cat command adds a trailing newline
				"echo_foobar.txt":      "foo\nbar\n",
				"echo_n_foobar.txt":    "foo\nbar\n",
				"echo_n_multiline.txt": "\n\nfoo\n  bar\nfoobar\n\nbaz\n\n",
				"var_multiline.txt":    "\n\nfoo\n  bar\nfoobar\n\nbaz\n\n\n",
				"var_catlines.txt":     "  foo   bar foobar  baz  \n",
				"var_enumfile.txt":     "0:\n1:\n2:foo\n3:  bar\n4:foobar\n5:\n6:baz\n7:\n8:\n",
			},
		}
		tt.Run(t)
	}
}

func TestSpecialVars(t *testing.T) {
	const dir = "testdata/special_vars"
	const target = "default"

	var buff bytes.Buffer
	e := &task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: target}))

	toAbs := func(rel string) string {
		abs, err := filepath.Abs(rel)
		assert.NoError(t, err)
		return abs
	}

	output := buff.String()

	// Root Taskfile
	assert.Contains(t, output, "root/TASK=print")
	assert.Contains(t, output, "root/ROOT_DIR="+toAbs("testdata/special_vars"))
	assert.Contains(t, output, "root/TASKFILE_DIR="+toAbs("testdata/special_vars"))

	// Included Taskfile
	assert.Contains(t, output, "included/TASK=included:print")
	assert.Contains(t, output, "included/ROOT_DIR="+toAbs("testdata/special_vars"))
	assert.Contains(t, output, "included/TASKFILE_DIR="+toAbs("testdata/special_vars/included"))
}

func TestVarsInvalidTmpl(t *testing.T) {
	const (
		dir         = "testdata/vars/v2"
		target      = "invalid-var-tmpl"
		expectError = "template: :1: unexpected EOF"
	)

	e := &task.Executor{
		Dir:    dir,
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	assert.NoError(t, e.Setup(), "e.Setup()")
	assert.EqualError(t, e.Run(context.Background(), taskfile.Call{Task: target}), expectError, "e.Run(target)")
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
	assert.NoError(t, e.Setup(), "e.Setup()")
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: target}), "e.Run(target)")
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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "default"}))

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
		Dir:     dir,
		TempDir: filepathext.SmartJoin(dir, ".task"),
		Stdout:  &buff,
		Stderr:  &buff,
		Silent:  true,
	}
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "gen-foo"}))
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "gen-bar"}))

	for _, f := range files {
		if _, err := os.Stat(filepathext.SmartJoin(dir, f)); err != nil {
			t.Errorf("File should exist: %v", err)
		}
	}

	e.Silent = false

	// all: not up-to-date
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "gen-foo"}))
	assert.Equal(t, "task: [gen-foo] touch foo.txt", strings.TrimSpace(buff.String()))
	buff.Reset()
	// status: not up-to-date
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "gen-foo"}))
	assert.Equal(t, "task: [gen-foo] touch foo.txt", strings.TrimSpace(buff.String()))
	buff.Reset()

	// sources: not up-to-date
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "gen-bar"}))
	assert.Equal(t, "task: [gen-bar] touch bar.txt", strings.TrimSpace(buff.String()))
	buff.Reset()
	// all: up-to-date
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "gen-bar"}))
	assert.Equal(t, `task: Task "gen-bar" is up to date`, strings.TrimSpace(buff.String()))
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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "foo"}))
	if buff.String() != "" {
		t.Errorf("Got Output when none was expected: %s", buff.String())
	}

	// A precondition that was not met
	assert.Error(t, e.Run(context.Background(), taskfile.Call{Task: "impossible"}))

	if buff.String() != "task: 1 != 0 obviously!\n" {
		t.Errorf("Wrong output message: %s", buff.String())
	}
	buff.Reset()

	// Calling a task with a precondition in a dependency fails the task
	assert.Error(t, e.Run(context.Background(), taskfile.Call{Task: "depends_on_impossible"}))

	if buff.String() != "task: 1 != 0 obviously!\n" {
		t.Errorf("Wrong output message: %s", buff.String())
	}
	buff.Reset()

	// Calling a task with a precondition in a cmd fails the task
	assert.Error(t, e.Run(context.Background(), taskfile.Call{Task: "executes_failing_task_as_cmd"}))
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

	var srcFile = filepathext.SmartJoin(dir, srcTask)

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
	assert.NoError(t, e.Setup())

	for _, theTask := range []string{relTask, absTask, fileWithSpaces} {
		var destFile = filepathext.SmartJoin(dir, theTask)
		var upToDate = fmt.Sprintf("task: Task \"%s\" is up to date\n", srcTask) +
			fmt.Sprintf("task: Task \"%s\" is up to date\n", theTask)

		// Run task for the first time.
		assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: theTask}))

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
		assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: theTask}))
		if buff.String() != upToDate {
			t.Errorf("Wrong output message: %s", buff.String())
		}
		buff.Reset()
	}
}

func TestStatusChecksum(t *testing.T) {
	const dir = "testdata/checksum"

	files := []string{
		"generated.txt",
		".task/checksum/build",
	}

	for _, f := range files {
		_ = os.Remove(filepathext.SmartJoin(dir, f))

		_, err := os.Stat(filepathext.SmartJoin(dir, f))
		assert.Error(t, err)
	}

	var buff bytes.Buffer
	e := task.Executor{
		Dir:     dir,
		TempDir: filepathext.SmartJoin(dir, ".task"),
		Stdout:  &buff,
		Stderr:  &buff,
	}
	assert.NoError(t, e.Setup())

	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "build"}))
	for _, f := range files {
		_, err := os.Stat(filepathext.SmartJoin(dir, f))
		assert.NoError(t, err)
	}

	buff.Reset()
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "build"}))
	assert.Equal(t, `task: Task "build" is up to date`+"\n", buff.String())
}

func TestAlias(t *testing.T) {
	const dir = "testdata/alias"

	data, err := os.ReadFile(filepathext.SmartJoin(dir, "alias.txt"))
	assert.NoError(t, err)

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "f"}))
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
	assert.NoError(t, e.Setup())
	assert.Error(t, e.Run(context.Background(), taskfile.Call{Task: "x"}))
	assert.Equal(t, "", buff.String())
}

func TestAliasSummary(t *testing.T) {
	const dir = "testdata/alias"

	data, err := os.ReadFile(filepathext.SmartJoin(dir, "alias-summary.txt"))
	assert.NoError(t, err)

	var buff bytes.Buffer
	e := task.Executor{
		Dir:     dir,
		Summary: true,
		Stdout:  &buff,
		Stderr:  &buff,
	}
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "f"}))
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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "foo"}))
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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "foo"}))
	assert.Contains(t, buff.String(), "foobar")
}

func TestLabelInStatus(t *testing.T) {
	const dir = "testdata/label_status"

	e := task.Executor{
		Dir: dir,
	}
	assert.NoError(t, e.Setup())
	err := e.Status(context.Background(), taskfile.Call{Task: "foo"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "foobar")
	}
}

func TestLabelWithVariableExpansion(t *testing.T) {
	const dir = "testdata/label_var"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "foo"}))
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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "foo"}))
	assert.Contains(t, buff.String(), "foobar")
}

func TestNoLabelInList(t *testing.T) {
	const dir = "testdata/label_list"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	assert.NoError(t, e.Setup())
	e.ListTasks(task.FilterOutInternal(), task.FilterOutNoDesc())
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

	assert.NoError(t, e.Setup())

	var title string
	e.ListTasks(task.FilterOutInternal())
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

	assert.NoError(t, e.Setup())
	e.ListTasks(task.FilterOutInternal(), task.FilterOutNoDesc())

	var title string
	assert.Contains(t, buff.String(), "foo")
	for _, title = range []string{
		"voo",
		"doo",
	} {
		assert.NotContains(t, buff.String(), title)
	}
}

func TestStatusVariables(t *testing.T) {
	const dir = "testdata/status_vars"

	_ = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))
	_ = os.Remove(filepathext.SmartJoin(dir, "generated.txt"))

	var buff bytes.Buffer
	e := task.Executor{
		Dir:     dir,
		TempDir: filepathext.SmartJoin(dir, ".task"),
		Stdout:  &buff,
		Stderr:  &buff,
		Silent:  false,
		Verbose: true,
	}
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "build"}))

	assert.Contains(t, buff.String(), "d41d8cd98f00b204e9800998ecf8427e")

	inf, err := os.Stat(filepathext.SmartJoin(dir, "source.txt"))
	assert.NoError(t, err)
	ts := fmt.Sprintf("%d", inf.ModTime().Unix())
	tf := inf.ModTime().String()

	assert.Contains(t, buff.String(), ts)
	assert.Contains(t, buff.String(), tf)
}

func TestInit(t *testing.T) {
	const dir = "testdata/init"
	var file = filepathext.SmartJoin(dir, "Taskfile.yaml")

	_ = os.Remove(file)
	if _, err := os.Stat(file); err == nil {
		t.Errorf("Taskfile.yaml should not exist")
	}

	if err := task.InitTaskfile(io.Discard, dir); err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Errorf("Taskfile.yaml should exist")
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
	assert.NoError(t, e.Setup())
	assert.IsType(t, &task.MaximumTaskCallExceededError{}, e.Run(context.Background(), taskfile.Call{Task: "task-1"}))
}

func TestTaskVersion(t *testing.T) {
	tests := []struct {
		Dir     string
		Version string
	}{
		{"testdata/version/v2", "2"},
	}

	for _, test := range tests {
		t.Run(test.Dir, func(t *testing.T) {
			e := task.Executor{
				Dir:    test.Dir,
				Stdout: io.Discard,
				Stderr: io.Discard,
			}
			assert.NoError(t, e.Setup())
			assert.Equal(t, test.Version, e.Taskfile.Version)
			assert.Equal(t, 2, len(e.Taskfile.Tasks))
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
	assert.NoError(t, e.Setup())

	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "task-should-pass"}))
	assert.Error(t, e.Run(context.Background(), taskfile.Call{Task: "task-should-fail"}))
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "cmd-should-pass"}))
	assert.Error(t, e.Run(context.Background(), taskfile.Call{Task: "cmd-should-fail"}))
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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "pwd"}))
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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "build"}))

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
		Dir:     dir,
		TempDir: filepathext.SmartJoin(dir, ".task"),
		Stdout:  io.Discard,
		Stderr:  io.Discard,
		Dry:     true,
	}
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "default"}))

	_, err := os.Stat(checksumFile)
	assert.Error(t, err, "checksum file should not exist")

	e.Dry = false
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "default"}))
	_, err = os.Stat(checksumFile)
	assert.NoError(t, err, "checksum file should exist")
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task: include cycle detected between")
}

func TestIncorrectVersionIncludes(t *testing.T) {
	const dir = "testdata/incorrect_includes"
	expectedError := "task: Import with additional parameters is only available starting on Taskfile version v3"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}

	assert.EqualError(t, e.Setup(), expectedError)
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task: Failed to parse testdata/includes_incorrect/incomplete.yml:")
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
		}}
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
	assert.Error(t, err)
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
	assert.Error(t, err)
	assert.Equal(t, expected, err.Error())
}

func TestIncludesFromCustomTaskfile(t *testing.T) {
	tt := fileContentTest{
		Dir:        "testdata/includes_yaml",
		Entrypoint: "Custom.ext",
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

	assert.NoError(t, e.Setup())

	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "common:pwd"}))
	assert.Contains(t, buff.String(), "testdata/includes_rel_path/common")

	buff.Reset()
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "included:common:pwd"}))
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
		{"included internal direct", "included:task-3", true, ""},
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
			assert.NoError(t, e.Setup())

			err := e.Run(context.Background(), taskfile.Call{Task: test.task})
			if test.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.expectedOutput, buff.String())
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
			assert.NoError(t, e.Setup())

			err := e.Run(context.Background(), taskfile.Call{Task: test.task})
			if test.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "task-with-summary"}, taskfile.Call{Task: "other-task-with-summary"}))

	data, err := os.ReadFile(filepathext.SmartJoin(dir, "task-with-summary.txt"))
	assert.NoError(t, err)

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

	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "whereami"}))

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

	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "whereami"}))

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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: target}))

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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: target}))

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

func TestDisplaysErrorOnUnsupportedVersion(t *testing.T) {
	e := task.Executor{
		Dir:    "testdata/version/v1",
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	err := e.Setup()
	assert.Error(t, err)
	assert.Equal(t, "task: Taskfile versions prior to v2 are not supported anymore", err.Error())

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
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "default"}))
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
	const dir = "testdata/dotenv/error_included_envs"
	const entry = "Taskfile.yml"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:        dir,
		Entrypoint: entry,
		Summary:    true,
		Stdout:     &buff,
		Stderr:     &buff,
	}

	err := e.Setup()
	assert.Error(t, err)
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

func TestExitImmediately(t *testing.T) {
	const dir = "testdata/exit_immediately"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}
	assert.NoError(t, e.Setup())

	assert.Error(t, e.Run(context.Background(), taskfile.Call{Task: "default"}))
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

func TestDeferredCmds(t *testing.T) {
	const dir = "testdata/deferred"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	assert.NoError(t, e.Setup())

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
	assert.Error(t, e.Run(context.Background(), taskfile.Call{Task: "task-2"}))
	assert.Contains(t, buff.String(), expectedOutputOrder)
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
			assert.NoError(t, e.Setup())
			assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "default"}))
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
	assert.NoError(t, e.Setup())

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
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "bye"}))
	t.Log(buff.String())
	assert.Equal(t, strings.TrimSpace(buff.String()), expectedOutputOrder)
}

func TestIncludedVars(t *testing.T) {
	const dir = "testdata/include_with_vars"
	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	assert.NoError(t, e.Setup())

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
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "task1"}))
	t.Log(buff.String())
	assert.Equal(t, strings.TrimSpace(buff.String()), expectedOutputOrder)
}

func TestErrorCode(t *testing.T) {
	const dir = "testdata/error_code"

	var buff bytes.Buffer
	e := &task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}
	assert.NoError(t, e.Setup())

	err := e.Run(context.Background(), taskfile.Call{Task: "test-exit-code"})
	assert.Error(t, err)
	casted, ok := err.(*task.TaskRunError)
	assert.True(t, ok, "cannot cast returned error to *task.TaskRunError")
	assert.Equal(t, 42, casted.ExitCode(), "unexpected exit code from task")
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
	assert.NoError(t, e.Setup())
	err := e.Run(context.Background(), taskfile.Call{Task: "default"})
	assert.NoError(t, err)
	assert.NotEqual(t, `task: Task "default" is up to date`, strings.TrimSpace(buff.String()))
	buff.Reset()
	err = e.Run(context.Background(), taskfile.Call{Task: "test-sym"})
	assert.NoError(t, err)
	assert.NotEqual(t, `task: Task "test-sym" is up to date`, strings.TrimSpace(buff.String()))
	buff.Reset()
	err = e.Run(context.Background(), taskfile.Call{Task: "default"})
	assert.NoError(t, err)
	assert.NotEqual(t, `task: Task "default" is up to date`, strings.TrimSpace(buff.String()))
	buff.Reset()
	err = e.Run(context.Background(), taskfile.Call{Task: "default"})
	assert.NoError(t, err)
	assert.Equal(t, `task: Task "default" is up to date`, strings.TrimSpace(buff.String()))
	buff.Reset()
	err = e.Run(context.Background(), taskfile.Call{Task: "reset"})
	assert.NoError(t, err)
	buff.Reset()
	err = os.RemoveAll(dir + "/.task")
	assert.NoError(t, err)
}

func TestFileWatcherInterval(t *testing.T) {
	const dir = "testdata/watcher_interval"
	expectedOutput := strings.TrimSpace(`
task: Started watching for tasks: default
task: [default] echo "Hello, World!"
Hello, World!
task: [default] echo "Hello, World!"
Hello, World!
	`)

	var buff bytes.Buffer
	e := &task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Watch:  true,
	}

	assert.NoError(t, e.Setup())
	buff.Reset()

	err := os.MkdirAll(filepathext.SmartJoin(dir, "src"), 0755)
	assert.NoError(t, err)

	err = os.WriteFile(filepathext.SmartJoin(dir, "src/a"), []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := e.Run(ctx, taskfile.Call{Task: "default"})
				if err != nil {
					return
				}
			}
		}
	}(ctx)

	time.Sleep(10 * time.Millisecond)
	err = os.WriteFile(filepathext.SmartJoin(dir, "src/a"), []byte("test updated"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(700 * time.Millisecond)
	cancel()
	assert.Equal(t, expectedOutput, strings.TrimSpace(buff.String()))
	buff.Reset()
	err = os.RemoveAll(filepathext.SmartJoin(dir, ".task"))
	assert.NoError(t, err)
	err = os.RemoveAll(filepathext.SmartJoin(dir, "src"))
	assert.NoError(t, err)
}
