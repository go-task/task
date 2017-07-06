package task_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-task/task"

	"github.com/stretchr/testify/assert"
)

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

func TestVars(t *testing.T) {
	const dir = "testdata/vars"

	files := []struct {
		file    string
		content string
	}{
		{"foo.txt", "foo"},
		{"bar.txt", "bar"},
		{"foo2.txt", "foo2"},
		{"bar2.txt", "bar2"},
		{"equal.txt", "foo=bar"},
	}

	for _, f := range files {
		_ = os.Remove(filepath.Join(dir, f.file))
	}

	e := &task.Executor{
		Dir:    dir,
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	assert.NoError(t, e.ReadTaskfile())
	assert.NoError(t, e.Run("default"))

	for _, f := range files {
		d, err := ioutil.ReadFile(filepath.Join(dir, f.file))
		if err != nil {
			t.Errorf("Error reading %s: %v", f.file, err)
		}
		s := string(d)
		s = strings.TrimSpace(s)

		if s != f.content {
			t.Errorf("File content should be %s but is %s", f.content, s)
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

func TestParams(t *testing.T) {
	const dir = "testdata/params"
	var files = []struct {
		file    string
		content string
	}{
		{"hello.txt", "Hello\n"},
		{"world.txt", "World\n"},
		{"exclamation.txt", "!\n"},
		{"dep1.txt", "Dependence1\n"},
		{"dep2.txt", "Dependence2\n"},
		{"spanish.txt", "¡Holla mundo!\n"},
	}

	for _, f := range files {
		_ = os.Remove(filepath.Join(dir, f.file))
	}

	e := task.Executor{
		Dir:    dir,
		Stdout: ioutil.Discard,
		Stderr: ioutil.Discard,
	}
	assert.NoError(t, e.ReadTaskfile())
	assert.NoError(t, e.Run("default"))

	for _, f := range files {
		content, err := ioutil.ReadFile(filepath.Join(dir, f.file))
		assert.NoError(t, err)
		assert.Equal(t, f.content, string(content))
	}
}
