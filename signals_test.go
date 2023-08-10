//go:build signals
// +build signals

// This file contains tests for signal handling on Unix.
// Based on code from https://github.com/marco-m/timeit
// Due to how signals work, for robustness we always spawn a separate process;
// we never send signals to the test process.

package task_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

var (
	SLEEPIT, _ = filepath.Abs("./bin/sleepit")
)

func TestSignalSentToProcessGroup(t *testing.T) {
	task, err := getTaskPath()
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]struct {
		args     []string
		sendSigs int
		want     []string
		notWant  []string
	}{
		// regression:
		// - child is terminated, immediately, by "context canceled" (another bug???)
		"child does not handle sigint: receives sigint and terminates immediately": {
			args:     []string{task, "--", SLEEPIT, "default", "-sleep=10s"},
			sendSigs: 1,
			want: []string{
				"sleepit: ready\n",
				"sleepit: work started\n",
				"task: Signal received: \"interrupt\"\n",
				// 130 = 128 + SIGINT
				"task: Failed to run task \"default\": exit status 130\n",
			},
			notWant: []string{
				"task: Failed to run task \"default\": context canceled\n",
			},
		},
		// 2 regressions:
		// - child receives 2 signals instead of 1
		// - child is terminated, immediately, by "context canceled" (another bug???)
		// TODO we need -cleanup=2s only to show reliably the bug; once the fix is committed,
		// we can use -cleanup=50ms to speed the test up
		"child intercepts sigint: receives sigint and does cleanup": {
			args:     []string{task, "--", SLEEPIT, "handle", "-sleep=10s", "-cleanup=2s"},
			sendSigs: 1,
			want: []string{
				"sleepit: ready\n",
				"sleepit: work started\n",
				"task: Signal received: \"interrupt\"\n",
				"sleepit: got signal=interrupt count=1\n",
				"sleepit: work canceled\n",
				"sleepit: cleanup started\n",
				"sleepit: cleanup done\n",
				"task: Failed to run task \"default\": exit status 3\n",
			},
			notWant: []string{
				"sleepit: got signal=interrupt count=2\n",
				"task: Failed to run task \"default\": context canceled\n",
			},
		},
		// regression: child receives 2 signal instead of 1 and thus terminates abruptly
		"child simulates terraform: receives 1 sigint and does cleanup": {
			args:     []string{task, "--", SLEEPIT, "handle", "-term-after=2", "-sleep=10s", "-cleanup=50ms"},
			sendSigs: 1,
			want: []string{
				"sleepit: ready\n",
				"sleepit: work started\n",
				"task: Signal received: \"interrupt\"\n",
				"sleepit: got signal=interrupt count=1\n",
				"sleepit: work canceled\n",
				"sleepit: cleanup started\n",
				"sleepit: cleanup done\n",
				"task: Failed to run task \"default\": exit status 3\n",
			},
			notWant: []string{
				"sleepit: got signal=interrupt count=2\n",
				"sleepit: cleanup canceled\n",
				"task: Failed to run task \"default\": exit status 4\n",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			sut := exec.Command(tc.args[0], tc.args[1:]...)
			sut.Stdout = &out
			sut.Stderr = &out
			sut.Dir = "testdata/ignore_signals"
			// Create a new process group by setting the process group ID of the child
			// to the child PID.
			// By default, the child would inherit the process group of the parent, but
			// we want to avoid this, to protect the parent (the test process) from the
			// signal that this test will send. More info in the comments below for
			// syscall.Kill().
			sut.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}

			if err := sut.Start(); err != nil {
				t.Fatalf("starting the SUT process: %v", err)
			}

			// After the child is started, we want to avoid a race condition where we send
			// it a signal before it had time to setup its own signal handlers. Sleeping
			// is way too flaky, instead we parse the child output until we get a line
			// that we know is printed after the signal handlers are installed...
			ready := false
			timeout := time.Duration(time.Second)
			start := time.Now()
			for time.Since(start) < timeout {
				if strings.Contains(out.String(), "sleepit: ready\n") {
					ready = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !ready {
				t.Fatalf("sleepit not ready after %v\n"+
					"additional information:\n"+
					"  output:\n%s",
					timeout, out.String())
			}

			// When we have a running program in a shell and type CTRL-C, the tty driver
			// will send a SIGINT signal to all the processes in the foreground process
			// group (see https://en.wikipedia.org/wiki/Process_group).
			//
			// Here we want to emulate this behavior: send SIGINT to the process group of
			// the test executable. Although Go for some reasons doesn't wrap the
			// killpg(2) system call, what works is using syscall.Kill(-PID, SIGINT),
			// where the negative PID means the corresponding process group. Note that
			// this negative PID works only as long as the caller of the kill(2) system
			// call has a different PID, which is the case for this test.
			for i := 1; i <= tc.sendSigs; i++ {
				if err := syscall.Kill(-sut.Process.Pid, syscall.SIGINT); err != nil {
					t.Fatalf("sending INT signal to the process group: %v", err)
				}
				time.Sleep(1 * time.Millisecond)
			}

			err := sut.Wait()

			var wantErr *exec.ExitError
			const wantExitStatus = 201
			if errors.As(err, &wantErr) {
				if wantErr.ExitCode() != wantExitStatus {
					t.Errorf(
						"waiting for child process: got exit status %v; want %d\n"+
							"additional information:\n"+
							"  process state: %q",
						wantErr.ExitCode(), wantExitStatus, wantErr.String())
				}
			} else {
				t.Errorf("waiting for child process: got unexpected error type %v (%T); want (%T)",
					err, err, wantErr)
			}

			gotLines := strings.SplitAfter(out.String(), "\n")
			notFound := listDifference(tc.want, gotLines)
			if len(notFound) > 0 {
				t.Errorf("\nwanted but not found:\n%v", notFound)
			}

			found := listIntersection(tc.notWant, gotLines)
			if len(found) > 0 {
				t.Errorf("\nunwanted but found:\n%v", found)
			}

			if len(notFound) > 0 || len(found) > 0 {
				t.Errorf("\noutput:\n%v", gotLines)
			}
		})
	}
}

func getTaskPath() (string, error) {
	if info, err := os.Stat("./bin/task"); err == nil {
		return info.Name(), nil
	}

	if path, err := exec.LookPath("task"); err == nil {
		return path, nil
	}

	return "", errors.New("task: \"task\" binary was not found!")
}

// Return the difference of the two lists: the elements that are present in the first
// list, but not in the second one. The notion of presence is not with `=` but with
// string.Contains(l2, l1).
// FIXME this does not enforce ordering. We might want to support both.
func listDifference(lines1, lines2 []string) []string {
	difference := []string{}
	for _, l1 := range lines1 {
		found := false
		for _, l2 := range lines2 {
			if strings.Contains(l2, l1) {
				found = true
				break
			}
		}
		if !found {
			difference = append(difference, l1)
		}
	}

	return difference
}

// Return the intersection of the two lists: the elements that are present in both lists.
// The notion of presence is not with '=' but with string.Contains(l2, l1)
// FIXME this does not enforce ordering. We might want to support both.
func listIntersection(lines1, lines2 []string) []string {
	intersection := []string{}
	for _, l1 := range lines1 {
		for _, l2 := range lines2 {
			if strings.Contains(l2, l1) {
				intersection = append(intersection, l1)
				break
			}
		}
	}

	return intersection
}
