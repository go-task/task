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
	if err := e.setCurrentDir(); err != nil {
		return err
	}

	if err := e.readTaskfile(); err != nil {
		return err
	}

	e.setupFuzzyModel()

	if err := e.setupTempDir(); err != nil {
		return err
	}
	e.setupStdFiles()
	e.setupLogger()
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
	if e.Dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		e.Dir = wd
	} else if !filepath.IsAbs(e.Dir) {
		abs, err := filepath.Abs(e.Dir)
		if err != nil {
			return err
		}
		e.Dir = abs
	}
	return nil
}

func (e *Executor) readTaskfile() error {
	var err error
	e.Taskfile, e.Dir, err = read.Taskfile(&read.ReaderNode{
		Dir:        e.Dir,
		Entrypoint: e.Entrypoint,
		Parent:     nil,
		Optional:   false,
	})
	return err
}

func (e *Executor) setupFuzzyModel() {
	if e.Taskfile != nil {
		return
	}

	model := fuzzy.NewModel()
	model.SetThreshold(1) // because we want to build grammar based on every task name

	var words []string
	for taskName := range e.Taskfile.Tasks {
		words = append(words, taskName)

		for _, task := range e.Taskfile.Tasks {
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
		Stdout:  e.Stdout,
		Stderr:  e.Stderr,
		Verbose: e.Verbose,
		Color:   e.Color,
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
		userWorkingDir, err := os.Getwd()
		if err != nil {
			return err
		}
		e.Compiler = &compilerv3.CompilerV3{
			Dir:            e.Dir,
			UserWorkingDir: userWorkingDir,
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
		if _, ok := e.Taskfile.Env.Mapping[key]; !ok {
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

	e.taskCallCount = make(map[string]*int32, len(e.Taskfile.Tasks))
	e.mkdirMutexMap = make(map[string]*sync.Mutex, len(e.Taskfile.Tasks))
	for k := range e.Taskfile.Tasks {
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
		return fmt.Errorf(`task: Taskfile versions prior to v2 are not supported anymore`)
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

		for _, task := range e.Taskfile.Tasks {
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
		for _, task := range e.Taskfile.Tasks {
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

		for _, task := range e.Taskfile.Tasks {
			if task.Run != "" {
				return errors.New(`task: Setting the "run" type is only available starting on Taskfile version v3.7`)
			}
		}
	}

	return nil
}
