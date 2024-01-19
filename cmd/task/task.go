package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"mvdan.cc/sh/v3/syntax"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/args"
	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/experiments"
	"github.com/go-task/task/v3/internal/flags"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/sort"
	ver "github.com/go-task/task/v3/internal/version"
	"github.com/go-task/task/v3/taskfile/ast"
)

func main() {
	if err := run(); err != nil {
		l := &logger.Logger{
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
			Verbose: flags.Verbose,
			Color:   flags.Color,
		}
		if err, ok := err.(*errors.TaskRunError); ok && flags.ExitCode {
			l.Errf(logger.Red, "%v\n", err)
			os.Exit(err.TaskExitCode())
		}
		if err, ok := err.(errors.TaskError); ok {
			l.Errf(logger.Red, "%v\n", err)
			os.Exit(err.Code())
		}
		l.Errf(logger.Red, "%v\n", err)
		os.Exit(errors.CodeUnknown)
	}
	os.Exit(errors.CodeOk)
}

func run() error {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	if err := flags.Validate(); err != nil {
		return err
	}

	dir := flags.Dir
	entrypoint := flags.Entrypoint

	if flags.Version {
		fmt.Printf("Task version: %s\n", ver.GetVersion())
		return nil
	}

	if flags.Help {
		pflag.Usage()
		return nil
	}

	if flags.Experiments {
		l := &logger.Logger{
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
			Verbose: flags.Verbose,
			Color:   flags.Color,
		}
		return experiments.List(l)
	}

	if flags.Init {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		if err := task.InitTaskfile(os.Stdout, wd); err != nil {
			log.Fatal(err)
		}
		return nil
	}

	if flags.Global {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("task: Failed to get user home directory: %w", err)
		}
		dir = home
	}

	if entrypoint != "" {
		dir = filepath.Dir(entrypoint)
		entrypoint = filepath.Base(entrypoint)
	}

	var taskSorter sort.TaskSorter
	switch flags.TaskSort {
	case "none":
		taskSorter = &sort.Noop{}
	case "alphanumeric":
		taskSorter = &sort.AlphaNumeric{}
	}

	e := task.Executor{
		Dir:         dir,
		Entrypoint:  entrypoint,
		Force:       flags.Force,
		ForceAll:    flags.ForceAll,
		Insecure:    flags.Insecure,
		Download:    flags.Download,
		Offline:     flags.Offline,
		Timeout:     flags.Timeout,
		Watch:       flags.Watch,
		Verbose:     flags.Verbose,
		Silent:      flags.Silent,
		AssumeYes:   flags.AssumeYes,
		Dry:         flags.Dry || flags.Status,
		Summary:     flags.Summary,
		Parallel:    flags.Parallel,
		Color:       flags.Color,
		Concurrency: flags.Concurrency,
		Interval:    flags.Interval,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,

		OutputStyle: flags.Output,
		TaskSorter:  taskSorter,
	}

	listOptions := task.NewListOptions(flags.List, flags.ListAll, flags.ListJson, flags.NoStatus)
	if err := listOptions.Validate(); err != nil {
		return err
	}

	if err := e.Setup(); err != nil {
		return err
	}

	// If the download flag is specified, we should stop execution as soon as
	// taskfile is downloaded
	if flags.Download {
		return nil
	}

	if (listOptions.ShouldListTasks()) && flags.Silent {
		return e.ListTaskNames(flags.ListAll)
	}

	if listOptions.ShouldListTasks() {
		foundTasks, err := e.ListTasks(listOptions)
		if err != nil {
			return err
		}
		if !foundTasks {
			os.Exit(errors.CodeUnknown)
		}
		return nil
	}

	var (
		calls   []*ast.Call
		globals *ast.Vars
	)

	tasksAndVars, cliArgs, err := getArgs()
	if err != nil {
		return err
	}

	calls, globals = args.Parse(tasksAndVars...)

	// If there are no calls, run the default task instead
	if len(calls) == 0 {
		calls = append(calls, &ast.Call{Task: "default"})
	}

	globals.Set("CLI_ARGS", ast.Var{Value: cliArgs})
	globals.Set("CLI_FORCE", ast.Var{Value: flags.Force || flags.ForceAll})
	e.Taskfile.Vars.Merge(globals)

	if !flags.Watch {
		e.InterceptInterruptSignals()
	}

	ctx := context.Background()

	if flags.Status {
		return e.Status(ctx, calls...)
	}

	return e.Run(ctx, calls...)
}

func getArgs() ([]string, string, error) {
	var (
		args          = pflag.Args()
		doubleDashPos = pflag.CommandLine.ArgsLenAtDash()
	)

	if doubleDashPos == -1 {
		return args, "", nil
	}

	var quotedCliArgs []string
	for _, arg := range args[doubleDashPos:] {
		quotedCliArg, err := syntax.Quote(arg, syntax.LangBash)
		if err != nil {
			return nil, "", err
		}
		quotedCliArgs = append(quotedCliArgs, quotedCliArg)
	}
	return args[:doubleDashPos], strings.Join(quotedCliArgs, " "), nil
}
