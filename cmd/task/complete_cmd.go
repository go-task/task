package main

import (
	"io"
	"os"

	"github.com/spf13/pflag"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/complete"
)

func runComplete(args []string) error {
	// Strip the completion-control flags the wrapper prepends; the rest is the
	// user's command line to complete.
	opts, args := complete.ParseOptions(args)

	dir, entrypoint, global := extractTaskfileFlags(args)

	e := task.NewExecutor(
		task.WithDir(dir),
		task.WithEntrypoint(entrypoint),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
		task.WithVersionCheck(false),
	)
	if global {
		if home, err := os.UserHomeDir(); err == nil {
			e.Options(task.WithDir(home))
		}
	}

	// Loading the Taskfile parses YAML (and may hit the network for remote
	// Taskfiles), so skip it entirely when completing flags or their values.
	// Best-effort: a missing or broken Taskfile must not break completion.
	if complete.NeedsTaskfile(args, pflag.CommandLine) {
		_ = e.Setup()
	}

	suggs, dirv := complete.Complete(e, pflag.CommandLine, args, opts)
	complete.Write(os.Stdout, suggs, dirv)
	return nil
}

func extractTaskfileFlags(args []string) (dir, entrypoint string, global bool) {
	fs := pflag.NewFlagSet("complete", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.ParseErrorsAllowlist.UnknownFlags = true
	fs.Usage = func() {}
	fs.StringVarP(&dir, "dir", "d", "", "")
	fs.StringVarP(&entrypoint, "taskfile", "t", "", "")
	fs.BoolVarP(&global, "global", "g", false, "")
	_ = fs.Parse(args)
	return
}
