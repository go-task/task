//go:build watch && !windows
// +build watch,!windows

package task_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestWatchProcessExitsOnSIGHUP(t *testing.T) {
	taskPath, err := findTaskBinaryForWatchTest()
	if err != nil {
		t.Skipf("skipping watcher signal test: %v", err)
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write seed source: %v", err)
	}

	taskfile := `version: '3'

tasks:
  default:
    watch: true
    sources:
      - src/**/*
    cmds:
      - echo "watch run"`
	if err := os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o644); err != nil {
		t.Fatalf("write taskfile: %v", err)
	}

	var out bytes.Buffer
	sut := exec.Command(taskPath, "--watch", "default")
	sut.Stdout = &out
	sut.Stderr = &out
	sut.Dir = dir
	if err := sut.Start(); err != nil {
		t.Fatalf("start task watcher process: %v", err)
	}

	if err := waitForOutputContains(&out, "Started watching for tasks: default", 5*time.Second); err != nil {
		_ = sut.Process.Kill()
		_, _ = sut.Process.Wait()
		t.Fatalf("watch process did not reach ready state: %v\noutput:\n%s", err, out.String())
	}

	if err := sut.Process.Signal(syscall.SIGHUP); err != nil {
		_ = sut.Process.Kill()
		_, _ = sut.Process.Wait()
		t.Fatalf("send SIGHUP to watch process: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- sut.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("watch process exit after SIGHUP: %v\noutput:\n%s", err, out.String())
		}
	case <-time.After(5 * time.Second):
		_ = sut.Process.Kill()
		t.Fatalf("watch process did not exit after SIGHUP\noutput:\n%s", out.String())
	}
}

func findTaskBinaryForWatchTest() (string, error) {
	if info, err := os.Stat("./bin/task"); err == nil {
		return info.Name(), nil
	}
	return exec.LookPath("task")
}

func waitForOutputContains(out *bytes.Buffer, needle string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), needle) {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return os.ErrDeadlineExceeded
}
