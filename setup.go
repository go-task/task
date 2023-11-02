package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/sajari/fuzzy"

	compilerv2 "github.com/go-task/task/v3/internal/compiler/v2"
	compilerv3 "github.com/go-task/task/v3/internal/compiler/v3"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/taskfile"
	"github.com/go-task/task/v3/taskfile/read"
)

func (e *Executor) Setup() error {
	e.setupLogger()
	if err := e.setCurrentDir(); err != nil {
		return err
	}
	if err := e.setupTempDir(); err != nil {
		return err
	}
	if err := e.readTaskfile(); err != nil {
		return err
	}
	e.setupFuzzyModel()
	e.setupStdFiles()
	if err := e.setupOutput(); err != nil {
		return err
	}
	if err := e.setupCompiler(); err != nil {
		return err
	}
	if err := e.readDotEnvFiles(); err != nil {
		return err
	}

	if err := e.doVersionChecks(); err != nil {
		return err
	}
	e.setupDefaults()
	e.setupConcurrencyState()

	return nil
}

func (e *Executor) setCurrentDir() error {
	// If the entrypoint is already set, we don't need to do anything
	if e.Entrypoint != "" {
		return nil
	}

	// Default the directory to the current working directory
	if e.Dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		e.Dir = wd
	}

	// Search for a taskfile
	root, err := read.ExistsWalk(e.Dir)
	if err != nil {
		return err
	}
	e.Dir = filepath.Dir(root)
	e.Entrypoint = filepath.Base(root)

	return nil
}

func (e *Executor) readTaskfile() error {
	uri := filepath.Join(e.Dir, e.Entrypoint)
	node, err := read.NewNode(uri, e.Insecure)
	if err != nil {
		return err
	}
	e.Taskfile, err = read.Taskfile(
		node,
		e.Insecure,
		e.Download,
		e.Offline,
		e.Timeout,
		e.TempDir,
		e.Logger,
	)
	if err != nil {
		return err
	}
	return nil
}

func (e *Executor) setupFuzzyModel() {
	if e.Taskfile != nil {
		return
	}

	model := fuzzy.NewModel()
	model.SetThreshold(1) // because we want to build grammar based on every task name

	var words []string
	for _, taskName := range e.Taskfile.Tasks.Keys() {
		words = append(words, taskName)

		for _, task := range e.Taskfile.Tasks.Values() {
			words = append(words, task.Aliases...)
		}
	}

	model.Train(words)
	e.fuzzyModel = model
}

func (e *Executor) setupTempDir() error {
	if e.TempDir != "" {
		return nil
	}

	if os.Getenv("TASK_TEMP_DIR") == "" {
		e.TempDir = filepathext.SmartJoin(e.Dir, ".task")
	} else if filepath.IsAbs(os.Getenv("TASK_TEMP_DIR")) || strings.HasPrefix(os.Getenv("TASK_TEMP_DIR"), "~") {
		tempDir, err := execext.Expand(os.Getenv("TASK_TEMP_DIR"))
		if err != nil {
			return err
		}
		projectDir, _ := filepath.Abs(e.Dir)
		projectName := filepath.Base(projectDir)
		e.TempDir = filepathext.SmartJoin(tempDir, projectName)
	} else {
		e.TempDir = filepathext.SmartJoin(e.Dir, os.Getenv("TASK_TEMP_DIR"))
	}

	return nil
}

func (e *Executor) setupStdFiles() {
	if e.Stdin == nil {
		e.Stdin = os.Stdin
	}
	if e.Stdout == nil {
		e.Stdout = os.Stdout
	}
	if e.Stderr == nil {
		e.Stderr = os.Stderr
	}
}

func (e *Executor) setupLogger() {
	e.Logger = &logger.Logger{
		Stdin:      e.Stdin,
		Stdout:     e.Stdout,
		Stderr:     e.Stderr,
		Verbose:    e.Verbose,
		Color:      e.Color,
		AssumeYes:  e.AssumeYes,
		AssumeTerm: e.AssumeTerm,
	}
}

func (e *Executor) setupOutput() error {
	if !e.OutputStyle.IsSet() {
		e.OutputStyle = e.Taskfile.Output
	}

	var err error
	e.Output, err = output.BuildFor(&e.OutputStyle)
	return err
}

func (e *Executor) setupCompiler() error {
	if e.Taskfile.Version.LessThan(taskfile.V3) {
		var err error
		e.taskvars, err = read.Taskvars(e.Dir)
		if err != nil {
			return err
		}

		e.Compiler = &compilerv2.CompilerV2{
			Dir:          e.Dir,
			Taskvars:     e.taskvars,
			TaskfileVars: e.Taskfile.Vars,
			Expansions:   e.Taskfile.Expansions,
			Logger:       e.Logger,
		}
	} else {
		if e.UserWorkingDir == "" {
			var err error
			e.UserWorkingDir, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		e.Compiler = &compilerv3.CompilerV3{
			Dir:            e.Dir,
			UserWorkingDir: e.UserWorkingDir,
			TaskfileEnv:    e.Taskfile.Env,
			TaskfileVars:   e.Taskfile.Vars,
			Logger:         e.Logger,
		}
	}

	return nil
}

func (e *Executor) readDotEnvFiles() error {
	if e.Taskfile.Version.LessThan(taskfile.V3) {
		return nil
	}

	env, err := read.Dotenv(e.Compiler, e.Taskfile, e.Dir)
	if err != nil {
		return err
	}

	err = env.Range(func(key string, value taskfile.Var) error {
		if ok := e.Taskfile.Env.Exists(key); !ok {
			e.Taskfile.Env.Set(key, value)
		}
		return nil
	})
	return err
}

func (e *Executor) setupDefaults() {
	// Color available only on v3
	if e.Taskfile.Version.LessThan(taskfile.V3) {
		e.Logger.Color = false
	}

	if e.Taskfile.Method == "" {
		if e.Taskfile.Version.Compare(taskfile.V3) >= 0 {
			e.Taskfile.Method = "checksum"
		} else {
			e.Taskfile.Method = "timestamp"
		}
	}

	if e.Taskfile.Run == "" {
		e.Taskfile.Run = "always"
	}
}

func (e *Executor) setupConcurrencyState() {
	e.executionHashes = make(map[string]context.Context)

	e.taskCallCount = make(map[string]*int32, e.Taskfile.Tasks.Len())
	e.mkdirMutexMap = make(map[string]*sync.Mutex, e.Taskfile.Tasks.Len())
	for _, k := range e.Taskfile.Tasks.Keys() {
		e.taskCallCount[k] = new(int32)
		e.mkdirMutexMap[k] = &sync.Mutex{}
	}

	if e.Concurrency > 0 {
		e.concurrencySemaphore = make(chan struct{}, e.Concurrency)
	}
}

func (e *Executor) doVersionChecks() error {
	// Copy the version to avoid modifying the original
	v := &semver.Version{}
	*v = *e.Taskfile.Version

	if v.LessThan(taskfile.V2) {
		return fmt.Errorf(`task: version 1 schemas are no longer supported`)
	}

	if v.LessThan(taskfile.V3) {
		e.Logger.Errf(logger.Yellow, "task: version 2 schemas are deprecated and will be removed in a future release\nSee https://github.com/go-task/task/issues/1197 for more details\n")
	}

	// consider as equal to the greater version if round
	if v.Equal(taskfile.V2) {
		v = semver.MustParse("2.6")
	}
	if v.Equal(taskfile.V3) {
		v = semver.MustParse("3.8")
	}

	if v.GreaterThan(semver.MustParse("3.8")) {
		return fmt.Errorf(`task: Taskfile versions greater than v3.8 not implemented in the version of Task`)
	}

	if v.LessThan(semver.MustParse("2.1")) && !e.Taskfile.Output.IsSet() {
		return fmt.Errorf(`task: Taskfile option "output" is only available starting on Taskfile version v2.1`)
	}
	if v.LessThan(semver.MustParse("2.2")) && e.Taskfile.Includes.Len() > 0 {
		return fmt.Errorf(`task: Including Taskfiles is only available starting on Taskfile version v2.2`)
	}
	if v.Compare(taskfile.V3) >= 0 && e.Taskfile.Expansions > 2 {
		return fmt.Errorf(`task: The "expansions" setting is not available anymore on v3.0`)
	}
	if v.LessThan(semver.MustParse("3.8")) && e.Taskfile.Output.Group.IsSet() {
		return fmt.Errorf(`task: Taskfile option "output.group" is only available starting on Taskfile version v3.8`)
	}

	if v.Compare(semver.MustParse("2.1")) <= 0 {
		err := errors.New(`task: Taskfile option "ignore_error" is only available starting on Taskfile version v2.1`)

		for _, task := range e.Taskfile.Tasks.Values() {
			if task.IgnoreError {
				return err
			}
			for _, cmd := range task.Cmds {
				if cmd.IgnoreError {
					return err
				}
			}
		}
	}

	if v.LessThan(semver.MustParse("2.6")) {
		for _, task := range e.Taskfile.Tasks.Values() {
			if len(task.Preconditions) > 0 {
				return errors.New(`task: Task option "preconditions" is only available starting on Taskfile version v2.6`)
			}
		}
	}

	if v.LessThan(taskfile.V3) {
		err := e.Taskfile.Includes.Range(func(_ string, taskfile taskfile.IncludedTaskfile) error {
			if taskfile.AdvancedImport {
				return errors.New(`task: Import with additional parameters is only available starting on Taskfile version v3`)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	if v.LessThan(semver.MustParse("3.7")) {
		if e.Taskfile.Run != "" {
			return errors.New(`task: Setting the "run" type is only available starting on Taskfile version v3.7`)
		}

		for _, task := range e.Taskfile.Tasks.Values() {
			if task.Run != "" {
				return errors.New(`task: Setting the "run" type is only available starting on Taskfile version v3.7`)
			}
		}
	}

	return nil
}
