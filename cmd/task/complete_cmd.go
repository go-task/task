package main

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/complete"
	"github.com/go-task/task/v3/internal/flags"
)

func runComplete(args []string) error {
	opts, args := complete.ParseOptions(args)

	// Overridden after WithFlags: a keystroke stays silent and never touches the
	// network, whatever the user typed.
	e := task.NewExecutor(
		flags.WithFlags(),
		task.WithStdout(io.Discard),
		task.WithStderr(io.Discard),
		task.WithStdin(strings.NewReader("")),
		task.WithVersionCheck(false),
		task.WithOffline(true),
		task.WithDownload(false),
	)

	// Best-effort: a missing or broken Taskfile must not break completion.
	if complete.NeedsTaskfile(args, pflag.CommandLine) {
		_ = e.Setup()
	}

	suggs, dirv := complete.Complete(e, pflag.CommandLine, args, opts)

	out := bufio.NewWriter(os.Stdout)
	complete.Write(out, suggs, dirv)
	return out.Flush()
}
