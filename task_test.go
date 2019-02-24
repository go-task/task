package task_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-task/task/v2"
	"github.com/go-task/task/v2/internal/taskfile"

	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
)

// fileContentTest provides a basic reusable test-case for running a Taskfile
// and inspect generated files.
type fileContentTest struct {
	Dir       string
	Target    string
	TrimSpace bool
	Files     map[string]string
}

func (fct fileContentTest) name(file string) string {
	return fmt.Sprintf("target=%q,file=%q", fct.Target, file)
}

func (fct fileContentTest) Run(t *testing.T) {
	for f := range fct.Files {
		_ = os.Remove(filepath.Join(fct.Dir, f))
	}

	e := &task.Executor{
		Dir:    fct.Dir,
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	assert.NoError(t, e.Setup(), "e.Setup()")
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: fct.Target}), "e.Run(target)")

	for name, expectContent := range fct.Files {
		t.Run(fct.name(name), func(t *testing.T) {
			b, err := ioutil.ReadFile(filepath.Join(fct.Dir, name))
			assert.NoError(t, err, "Error reading file")
			s := string(b)
			if fct.TrimSpace {
				s = strings.TrimSpace(s)
			}
			assert.Equal(t, expectContent, s, "unexpected file content")
		})
	}
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

func TestVarsV1(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/vars/v1",
		Target:    "default",
		TrimSpace: true,
		Files: map[string]string{
			// hello task:
			"foo.txt":              "foo",
			"bar.txt":              "bar",
			"baz.txt":              "baz",
			"tmpl_foo.txt":         "foo",
			"tmpl_bar.txt":         "<no value>",
			"tmpl_foo2.txt":        "foo2",
			"tmpl_bar2.txt":        "bar2",
			"shtmpl_foo.txt":       "foo",
			"shtmpl_foo2.txt":      "foo2",
			"nestedtmpl_foo.txt":   "{{.FOO}}",
			"nestedtmpl_foo2.txt":  "foo2",
			"foo2.txt":             "foo2",
			"bar2.txt":             "bar2",
			"baz2.txt":             "baz2",
			"tmpl2_foo.txt":        "<no value>",
			"tmpl2_foo2.txt":       "foo2",
			"tmpl2_bar.txt":        "<no value>",
			"tmpl2_bar2.txt":       "<no value>",
			"shtmpl2_foo.txt":      "<no value>",
			"shtmpl2_foo2.txt":     "foo2",
			"nestedtmpl2_foo2.txt": "{{.FOO2}}",
			"override.txt":         "bar",
		},
	}
	tt.Run(t)
	// Ensure identical results when running hello task directly.
	tt.Target = "hello"
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
		},
	}
	tt.Run(t)
	// Ensure identical results when running hello task directly.
	tt.Target = "hello"
	tt.Run(t)
}

func TestMultilineVars(t *testing.T) {
	for _, dir := range []string{"testdata/vars/v1/multiline", "testdata/vars/v2/multiline"} {
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

func TestVarsInvalidTmpl(t *testing.T) {
	const (
		dir         = "testdata/vars/v1"
		target      = "invalid-var-tmpl"
		expectError = "template: :1: unexpected EOF"
	)

	e := &task.Executor{
		Dir:    dir,
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	assert.NoError(t, e.Setup(), "e.Setup()")
	assert.EqualError(t, e.Run(context.Background(), taskfile.Call{Task: target}), expectError, "e.Run(target)")
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
		_ = os.Remove(filepath.Join(dir, f))
	}

	e := &task.Executor{
		Dir:    dir,
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "default"}))

	for _, f := range files {
		f = filepath.Join(dir, f)
		if _, err := os.Stat(f); err != nil {
			t.Errorf("File %s should exists", f)
		}
	}
}

func TestStatus(t *testing.T) {
	const dir = "testdata/status"
	var file = filepath.Join(dir, "foo.txt")

	_ = os.Remove(file)

	if _, err := os.Stat(file); err == nil {
		t.Errorf("File should not exists: %v", err)
	}

	var buff bytes.Buffer
	e := &task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		Silent: true,
	}
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "gen-foo"}))

	if _, err := os.Stat(file); err != nil {
		t.Errorf("File should exists: %v", err)
	}

	e.Silent = false
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "gen-foo"}))

	if buff.String() != `task: Task "gen-foo" is up to date`+"\n" {
		t.Errorf("Wrong output message: %s", buff.String())
	}
}

func TestGenerates(t *testing.T) {
	const (
		srcTask        = "sub/src.txt"
		relTask        = "rel.txt"
		absTask        = "abs.txt"
		fileWithSpaces = "my text file.txt"
	)

	// This test does not work with a relative dir.
	dir, err := filepath.Abs("testdata/generates")
	assert.NoError(t, err)
	var srcFile = filepath.Join(dir, srcTask)

	for _, task := range []string{srcTask, relTask, absTask, fileWithSpaces} {
		path := filepath.Join(dir, task)
		_ = os.Remove(path)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("File should not exists: %v", err)
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
		var destFile = filepath.Join(dir, theTask)
		var upToDate = fmt.Sprintf("task: Task \"%s\" is up to date\n", srcTask) +
			fmt.Sprintf("task: Task \"%s\" is up to date\n", theTask)

		// Run task for the first time.
		assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: theTask}))

		if _, err := os.Stat(srcFile); err != nil {
			t.Errorf("File should exists: %v", err)
		}
		if _, err := os.Stat(destFile); err != nil {
			t.Errorf("File should exists: %v", err)
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
		_ = os.Remove(filepath.Join(dir, f))

		_, err := os.Stat(filepath.Join(dir, f))
		assert.Error(t, err)
	}

	var buff bytes.Buffer
	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
	}
	assert.NoError(t, e.Setup())

	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "build"}))
	for _, f := range files {
		_, err := os.Stat(filepath.Join(dir, f))
		assert.NoError(t, err)
	}

	buff.Reset()
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "build"}))
	assert.Equal(t, `task: Task "build" is up to date`+"\n", buff.String())
}

func TestInit(t *testing.T) {
	const dir = "testdata/init"
	var file = filepath.Join(dir, "Taskfile.yml")

	_ = os.Remove(file)
	if _, err := os.Stat(file); err == nil {
		t.Errorf("Taskfile.yml should not exists")
	}

	if err := task.InitTaskfile(ioutil.Discard, dir); err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Errorf("Taskfile.yml should exists")
	}
}

func TestCyclicDep(t *testing.T) {
	const dir = "testdata/cyclic"

	e := task.Executor{
		Dir:    dir,
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	assert.NoError(t, e.Setup())
	assert.IsType(t, &task.MaximumTaskCallExceededError{}, e.Run(context.Background(), taskfile.Call{Task: "task-1"}))
}

func TestTaskVersion(t *testing.T) {
	tests := []struct {
		Dir     string
		Version string
	}{
		{"testdata/version/v1", "1"},
		{"testdata/version/v2", "2"},
	}

	for _, test := range tests {
		t.Run(test.Dir, func(t *testing.T) {
			e := task.Executor{
				Dir:    test.Dir,
				Stdout: ioutil.Discard,
				Stderr: ioutil.Discard,
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
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	assert.NoError(t, e.Setup())

	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "task-should-pass"}))
	assert.Error(t, e.Run(context.Background(), taskfile.Call{Task: "task-should-fail"}))
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "cmd-should-pass"}))
	assert.Error(t, e.Run(context.Background(), taskfile.Call{Task: "cmd-should-fail"}))
}

func TestExpand(t *testing.T) {
	const dir = "testdata/expand"

	home, err := homedir.Dir()
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

	file := filepath.Join(dir, "file.txt")
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

	assert.Equal(t, "touch file.txt", strings.TrimSpace(buff.String()))
	if _, err := os.Stat(file); err == nil {
		t.Errorf("File should not exist %s", file)
	}
}

// TestDryChecksum tests if the checksum file is not being written to disk
// if the dry mode is enabled.
func TestDryChecksum(t *testing.T) {
	const dir = "testdata/dry_checksum"

	checksumFile := filepath.Join(dir, ".task/checksum/default")
	_ = os.Remove(checksumFile)

	e := task.Executor{
		Dir:    dir,
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
		Dry:    true,
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
			"main.txt":               "main",
			"included_directory.txt": "included_directory",
			"included_taskfile.txt":  "included_taskfile",
		},
	}
	tt.Run(t)
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

func TestDetailsParsing(t *testing.T) {
	const dir = "testdata/details"

	e := task.Executor{
		Dir: dir,
	}
	assert.NoError(t, e.Setup())

	assert.Equal(t, e.Taskfile.Tasks["task-with-details"].Details, "details of task-with-details - line 1\nline 2\nline 3\n")
	assert.Equal(t, e.Taskfile.Tasks["other-task-with-details"].Details, "details of other-task-with-details")
	assert.Equal(t, e.Taskfile.Tasks["task-without-details"].Details, "")
}

func TestDetails(t *testing.T) {
	const dir = "testdata/details"

	var buff bytes.Buffer
	e := task.Executor{
		Dir:     dir,
		Stdout:  &buff,
		Stderr:  &buff,
		Details: true,
		Silent:  true,
	}
	assert.NoError(t, e.Setup())

	buff.Reset()
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "task-with-details"}))
	assert.Equal(t, buff.String(), "task: task-with-details\n\ndetails of task-with-details - line 1\n"+"line 2\n"+"line 3\n\nCommands:\n")

	assert.NotContains(t, buff.String(), "task-with-details was executed")
	assert.NotContains(t, buff.String(), "dependend-task was executed")

	buff.Reset()
	const noDetails = "task-without-details"
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: noDetails}))
	assert.Equal(t, buff.String(), "task: There is no detailed description for task: "+noDetails+"\n")

	buff.Reset()
	const firstTask = "other-task-with-details"
	const secondTask = "task-with-details"
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: firstTask}, taskfile.Call{Task: secondTask}))
	assert.Contains(t, buff.String(), "details of "+firstTask)
	assert.NotContains(t, buff.String(), "details of "+secondTask)

	buff.Reset()
	assert.NoError(t, e.Run(context.Background(), taskfile.Call{Task: "task-with-description-containing-empty-line"}))
	assert.Equal(t, buff.String(), "task: task-with-description-containing-empty-line\n\nFirst line followed by empty line\n\nLast Line\n\nCommands:\n")

}
