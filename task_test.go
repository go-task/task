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
	"github.com/go-task/task/internal/taskfile"

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
	assert.NoError(t, e.Run(taskfile.Call{Task: fct.Target}), "e.Run(target)")

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
			"env.txt": "GOOS='linux' GOARCH='amd64' CGO_ENABLED='0'\n",
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
	assert.EqualError(t, e.Run(taskfile.Call{Task: target}), expectError, "e.Run(target)")
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
	assert.NoError(t, e.Run(taskfile.Call{Task: "default"}))

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
	assert.NoError(t, e.Run(taskfile.Call{Task: "gen-foo"}))

	if _, err := os.Stat(file); err != nil {
		t.Errorf("File should exists: %v", err)
	}

	e.Silent = false
	assert.NoError(t, e.Run(taskfile.Call{Task: "gen-foo"}))

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
	assert.NoError(t, e.Setup())

	for _, theTask := range []string{relTask, absTask} {
		var destFile = filepath.Join(dir, theTask)
		var upToDate = fmt.Sprintf("task: Task \"%s\" is up to date\n", srcTask) +
			fmt.Sprintf("task: Task \"%s\" is up to date\n", theTask)

		// Run task for the first time.
		assert.NoError(t, e.Run(taskfile.Call{Task: theTask}))

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
		assert.NoError(t, e.Run(taskfile.Call{Task: theTask}))
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

	assert.NoError(t, e.Run(taskfile.Call{Task: "build"}))
	for _, f := range files {
		_, err := os.Stat(filepath.Join(dir, f))
		assert.NoError(t, err)
	}

	buff.Reset()
	assert.NoError(t, e.Run(taskfile.Call{Task: "build"}))
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
	assert.IsType(t, &task.MaximumTaskCallExceededError{}, e.Run(taskfile.Call{Task: "task-1"}))
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
	assert.NoError(t, e.Run(taskfile.Call{Task: "pwd"}))
	assert.Equal(t, home, strings.TrimSpace(buff.String()))
}

func TestDryRun(t *testing.T) {
	const dir = "testdata/dryrun"

	file := filepath.Join(dir, "file.txt")
	_ = os.Remove(file)

	var buff bytes.Buffer

	e := task.Executor{
		Dir:    dir,
		Stdout: &buff,
		Stderr: &buff,
		DryRun: true,
	}
	assert.NoError(t, e.Setup())
	assert.NoError(t, e.Run(taskfile.Call{Task: "build"}))

	if _, err := os.Stat(file); err == nil {
		t.Errorf("File should not exist %s", file)
	}
}
