//go:build test_e2e
// +build test_e2e

package main

import (
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"task": main_,
	}))
}

func TestE2E(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:   "testdata",
		Cmds:  map[string]func(ts *testscript.TestScript, neg bool, args []string){},
		Setup: func(e *testscript.Env) error { return nil },
	})
}
