package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/args"
	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/flags"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/version"
	"github.com/go-task/task/v3/taskfile"
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
	log := &logger.Logger{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Verbose: flags.Verbose,
		Color:   flags.Color,
	}

	if err := flags.Validate(); err != nil {
		return err
	}

	if err := experiments.Validate(); err != nil {
		log.Warnf("%s\n", err.Error())
	}

	if flags.Version {
		fmt.Println(version.GetVersionWithBuildInfo())
		return nil
	}

	if flags.Help {
		pflag.Usage()
		return nil
	}

	if flags.Experiments {
		return log.PrintExperiments()
	}

	if flags.Init {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		args, _, err := args.Get()
		if err != nil {
			return err
		}
		path := wd
		if len(args) > 0 {
			name := args[0]
			if filepathext.IsExtOnly(name) {
				name = filepathext.SmartJoin(filepath.Dir(name), "Taskfile"+filepath.Ext(name))
			}
			path = filepathext.SmartJoin(wd, name)
		}
		finalPath, err := task.InitTaskfile(path)
		if err != nil {
			return err
		}
		if !flags.Silent {
			if flags.Verbose {
				log.Outf(logger.Default, "%s\n", task.DefaultTaskfile)
			}
			log.Outf(logger.Green, "Taskfile created: %s\n", filepathext.TryAbsToRel(finalPath))
		}
		return nil
	}

	if flags.Completion != "" {
		script, err := task.Completion(flags.Completion)
		if err != nil {
			return err
		}
		fmt.Println(script)
		return nil
	}

	if err := experiments.Validate(); err != nil {
		log.Warnf("%s\n", err.Error())
	}

	// Create a new root node for the given entrypoint
	node, err := taskfile.NewRootNode(
		flags.Entrypoint,
		flags.Dir,
		flags.Insecure,
	)
	if err != nil {
		return err
	}

	tempDir, err := task.NewTempDir(node.Dir())
	if err != nil {
		return err
	}

	reader := taskfile.NewReader(
		flags.WithFlags(),
		taskfile.WithTempDir(tempDir.Remote),
		taskfile.WithDebugFunc(func(s string) {
			log.VerboseOutf(logger.Magenta, s)
		}),
		taskfile.WithPromptFunc(func(s string) error {
			return log.Prompt(logger.Yellow, s, "n", "y", "yes")
		}),
	)

	ctx, cf := context.WithTimeout(context.Background(), flags.Timeout)
	defer cf()
	graph, err := reader.Read(ctx, node)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return &errors.TaskfileNetworkTimeoutError{URI: node.Location(), Timeout: flags.Timeout}
		}
		return err
	}

	executor, err := task.NewExecutor(graph,
		flags.WithFlags(),
		task.WithDir(node.Dir()),
		task.WithTempDir(tempDir),
	)
	if err != nil {
		return err
	}

	// If the download flag is specified, we should stop execution as soon as
	// taskfile is downloaded
	if flags.Download {
		return nil
	}

	if flags.ClearCache {
		cachePath := filepath.Join(executor.TempDir.Remote, "remote")
		return os.RemoveAll(cachePath)
	}

	listOptions := task.NewListOptions(
		flags.List,
		flags.ListAll,
		flags.ListJson,
		flags.NoStatus,
	)
	if listOptions.ShouldListTasks() {
		if flags.Silent {
			return executor.ListTaskNames(flags.ListAll)
		}
		foundTasks, err := executor.ListTasks(listOptions)
		if err != nil {
			return err
		}
		if !foundTasks {
			os.Exit(errors.CodeUnknown)
		}
		return nil
	}

	// Parse the remaining arguments
	cliArgsPreDash, cliArgsPostDash, err := args.Get()
	if err != nil {
		return err
	}
	calls, globals := args.Parse(cliArgsPreDash...)

	// If there are no calls, run the default task instead
	if len(calls) == 0 {
		calls = append(calls, &task.Call{Task: "default"})
	}

	cliArgsPostDashQuoted, err := args.ToQuotedString(cliArgsPostDash)
	if err != nil {
		return err
	}
	globals.Set("CLI_ARGS", ast.Var{Value: cliArgsPostDashQuoted})
	globals.Set("CLI_ARGS_LIST", ast.Var{Value: cliArgsPostDash})
	globals.Set("CLI_FORCE", ast.Var{Value: flags.Force || flags.ForceAll})
	globals.Set("CLI_SILENT", ast.Var{Value: flags.Silent})
	globals.Set("CLI_VERBOSE", ast.Var{Value: flags.Verbose})
	globals.Set("CLI_OFFLINE", ast.Var{Value: flags.Offline})
	executor.Taskfile.Vars.Merge(globals, nil)

	if !flags.Watch {
		executor.InterceptInterruptSignals()
	}

	ctx = context.Background()

	if flags.Status {
		return executor.Status(ctx, calls...)
	}

	return executor.Run(ctx, calls...)
}
