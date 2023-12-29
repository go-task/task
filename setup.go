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

	"github.com/go-task/task/v3/internal/compiler"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/taskfile"
	"github.com/go-task/task/v3/taskfile/ast"
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
	root, err := taskfile.ExistsWalk(e.Dir)
	if err != nil {
		return err
	}
	e.Dir = filepath.Dir(root)
	e.Entrypoint = filepath.Base(root)

	return nil
}

func (e *Executor) readTaskfile() error {
	uri := filepath.Join(e.Dir, e.Entrypoint)
	node, err := taskfile.NewNode(uri, e.Insecure)
	if err != nil {
		return err
	}
	e.Taskfile, err = taskfile.Read(
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
	if e.UserWorkingDir == "" {
		var err error
		e.UserWorkingDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	e.Compiler = &compiler.Compiler{
		Dir:            e.Dir,
		UserWorkingDir: e.UserWorkingDir,
		TaskfileEnv:    e.Taskfile.Env,
		TaskfileVars:   e.Taskfile.Vars,
		Logger:         e.Logger,
	}
	return nil
}

func (e *Executor) readDotEnvFiles() error {
	if e.Taskfile.Version.LessThan(ast.V3) {
		return nil
	}

	env, err := taskfile.Dotenv(e.Compiler, e.Taskfile, e.Dir)
	if err != nil {
		return err
	}

	err = env.Range(func(key string, value ast.Var) error {
		if ok := e.Taskfile.Env.Exists(key); !ok {
			e.Taskfile.Env.Set(key, value)
		}
		return nil
	})
	return err
}

func (e *Executor) setupDefaults() {
	if e.Taskfile.Method == "" {
		e.Taskfile.Method = "checksum"
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

	if v.LessThan(ast.V3) {
		return fmt.Errorf(`task: Taskfile schemas prior to v3 are no longer supported`)
	}

	// consider as equal to the greater version if round
	if v.Equal(ast.V3) {
		v = semver.MustParse("3.8")
	}

	if v.GreaterThan(semver.MustParse("3.8")) {
		return fmt.Errorf(`task: Taskfile versions greater than v3.8 not implemented in the version of Task`)
	}

	if v.LessThan(semver.MustParse("3.8")) && e.Taskfile.Output.Group.IsSet() {
		return fmt.Errorf(`task: Taskfile option "output.group" is only available starting on Taskfile version v3.8`)
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
