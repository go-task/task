package main

import (
	"io"
	"os"

	"github.com/spf13/pflag"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/complete"
)

func runComplete(args []string) error {
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

	// Best-effort: a missing or broken Taskfile must not break completion.
	_ = e.Setup()

	suggs, dirv := complete.Complete(e, pflag.CommandLine, args)
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
