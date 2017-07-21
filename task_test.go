package task_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-task/task"

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
	assert.NoError(t, e.ReadTaskfile(), "e.ReadTaskfile()")
	assert.NoError(t, e.Run(fct.Target), "e.Run(target)")

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

func TestVars(t *testing.T) {
	tt := fileContentTest{
		Dir:       "testdata/vars",
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
			"equal.txt":            "foo=bar",
			"override.txt":         "bar",
		},
	}
	tt.Run(t)
	// Ensure identical results when running hello task directly.
	tt.Target = "hello"
	tt.Run(t)
}

func TestVarsInvalidTmpl(t *testing.T) {
	const (
		dir         = "testdata/vars"
		target      = "invalid-var-tmpl"
		expectError = "template: :1: unexpected EOF"
	)

	e := &task.Executor{
		Dir:    dir,
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	assert.NoError(t, e.ReadTaskfile(), "e.ReadTaskfile()")
	assert.EqualError(t, e.Run(target), expectError, "e.Run(target)")
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
	assert.NoError(t, e.ReadTaskfile())
	assert.NoError(t, e.Run("default"))

	for _, f := range files {
		f = filepath.Join(dir, f)
		if _, err := os.Stat(f); err != nil {
			t.Errorf("File %s should exists", f)
		}
	}
}

func TestTaskCall(t *testing.T) {
	const dir = "testdata/task_call"

	files := []string{
		"foo.txt",
		"bar.txt",
	}

	for _, f := range files {
		_ = os.Remove(filepath.Join(dir, f))
	}

	e := &task.Executor{
		Dir:    dir,
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	assert.NoError(t, e.ReadTaskfile())
	assert.NoError(t, e.Run("default"))

	for _, f := range files {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Error(err)
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

	e := &task.Executor{
		Dir:    dir,
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	assert.NoError(t, e.ReadTaskfile())
	assert.NoError(t, e.Run("gen-foo"))

	if _, err := os.Stat(file); err != nil {
		t.Errorf("File should exists: %v", err)
	}

	buff := bytes.NewBuffer(nil)
	e.Stdout, e.Stderr = buff, buff
	assert.NoError(t, e.Run("gen-foo"))

	if buff.String() != `task: Task "gen-foo" is up to date`+"\n" {
		t.Errorf("Wrong output message: %s", buff.String())
	}
}

func TestGenerates(t *testing.T) {
	var srcTask = "sub/src.txt"
	var relTask = "rel.txt"
	var absTask = "abs.txt"

	// This test does not work with a relative dir.
	dir, err := filepath.Abs("testdata/generates")
	assert.NoError(t, err)
	var srcFile = filepath.Join(dir, srcTask)

	for _, task := range []string{srcTask, relTask, absTask} {
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
	assert.NoError(t, e.ReadTaskfile())

	for _, task := range []string{relTask, absTask} {
		var destFile = filepath.Join(dir, task)
		var upToDate = fmt.Sprintf("task: Task \"%s\" is up to date\n", srcTask) +
			fmt.Sprintf("task: Task \"%s\" is up to date\n", task)

		// Run task for the first time.
		assert.NoError(t, e.Run(task))

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
		assert.NoError(t, e.Run(task))
		if buff.String() != upToDate {
			t.Errorf("Wrong output message: %s", buff.String())
		}
		buff.Reset()
	}
}

func TestInit(t *testing.T) {
	const dir = "testdata/init"
	var file = filepath.Join(dir, "Taskfile.yml")

	_ = os.Remove(file)
	if _, err := os.Stat(file); err == nil {
		t.Errorf("Taskfile.yml should not exists")
	}

	if err := task.InitTaskfile(dir); err != nil {
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
	assert.NoError(t, e.ReadTaskfile())
	assert.IsType(t, &task.MaximumTaskCallExceededError{}, e.Run("task-1"))
}
