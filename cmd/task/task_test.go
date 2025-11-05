//go:build test_e2e
// +build test_e2e

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"task": main_,
	}))
}

func TestE2E(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"sleep": sleep,
			"touch": touch,
		},
		Setup: func(e *testscript.Env) error { return nil },
	})
}

func sleep(ts *testscript.TestScript, neg bool, args []string) {
	duration := time.Second
	if len(args) == 1 {
		d, err := time.ParseDuration(args[0])
		ts.Check(err)
		duration = d
	}
	time.Sleep(duration)
}

func touch(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 {
		ts.Fatalf("touch <file>")
	}
	// Get the relative path to the scripts current directory.
	path := ts.MkAbs(args[0])
	// Create the file (if necessary).
	err := os.MkdirAll(filepath.Dir(path), 0750)
	ts.Check(err)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	ts.Check(err)
	err = file.Close()
	ts.Check(err)
	// Now update the timestamp.
	t := time.Now()
	err = os.Chtimes(path, t, t)
	ts.Check(err)
}
