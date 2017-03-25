package task_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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

	c := exec.Command("task")
	c.Dir = dir
	if err := c.Run(); err != nil {
		t.Error(err)
		return
	}

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

	c := exec.Command("task")
	c.Dir = dir

	if err := c.Run(); err != nil {
		t.Error(err)
		return
	}

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

	c := exec.Command("task")
	c.Dir = dir

	if err := c.Run(); err != nil {
		t.Error(err)
		return
	}

	for _, f := range files {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Error(err)
		}
	}
}
