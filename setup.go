package task

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/sajari/fuzzy"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/fsext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/output"
	"github.com/go-task/task/v3/internal/version"
	"github.com/go-task/task/v3/taskfile"
	"github.com/go-task/task/v3/taskfile/ast"
)

func (e *Executor) Setup() error {
	e.setupLogger()
	node, err := e.getRootNode()
	if err != nil {
		return err
	}
	if err := e.setupTempDir(); err != nil {
		return err
	}
	if err := e.readTaskfile(node); err != nil {
		return err
	}
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

func (e *Executor) getRootNode() (taskfile.Node, error) {
	node, err := taskfile.NewRootNode(e.Entrypoint, e.Dir, e.Insecure, e.Timeout)
	if os.IsNotExist(err) {
		return nil, errors.TaskfileNotFoundError{
			URI:     fsext.DefaultDir(e.Entrypoint, e.Dir),
			Walk:    true,
			AskInit: true,
		}
	}
	if err != nil {
		return nil, err
	}
	e.Dir = node.Dir()
	return node, err
}

func (e *Executor) readTaskfile(node taskfile.Node) error {
	ctx, cf := context.WithTimeout(context.Background(), e.Timeout)
	defer cf()
	debugFunc := func(s string) {
		e.Logger.VerboseOutf(logger.Magenta, s)
	}
	promptFunc := func(s string) error {
		return e.Logger.Prompt(logger.Yellow, s, "n", "y", "yes")
	}
	reader := taskfile.NewReader(
		taskfile.WithInsecure(e.Insecure),
		taskfile.WithDownload(e.Download),
		taskfile.WithOffline(e.Offline),
		taskfile.WithTrustedHosts(e.TrustedHosts),
		taskfile.WithTempDir(e.TempDir.Remote),
		taskfile.WithCacheExpiryDuration(e.CacheExpiryDuration),
		taskfile.WithDebugFunc(debugFunc),
		taskfile.WithPromptFunc(promptFunc),
	)
	graph, err := reader.Read(ctx, node)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return &errors.TaskfileNetworkTimeoutError{URI: node.Location(), Timeout: e.Timeout}
		}
		return err
	}
	if e.Taskfile, err = graph.Merge(); err != nil {
		return err
	}
	return nil
}

func (e *Executor) setupFuzzyModel() {
	if e.Taskfile == nil {
		return
	}

	model := fuzzy.NewModel()
	model.SetThreshold(1) // because we want to build grammar based on every task name

	var words []string
	for name, task := range e.Taskfile.Tasks.All(nil) {
		if task.Internal {
			continue
		}
		words = append(words, name)
		words = slices.Concat(words, task.Aliases)
	}

	model.Train(words)
	e.fuzzyModel = model
}

func (e *Executor) setupTempDir() error {
	if e.TempDir != (TempDir{}) {
		return nil
	}

	tempDir := env.GetTaskEnv("TEMP_DIR")
	if tempDir == "" {
		e.TempDir = TempDir{
			Remote:      filepathext.SmartJoin(e.Dir, ".task"),
			Fingerprint: filepathext.SmartJoin(e.Dir, ".task"),
		}
	} else if filepath.IsAbs(tempDir) || strings.HasPrefix(tempDir, "~") {
		tempDir, err := execext.ExpandLiteral(tempDir)
		if err != nil {
			return err
		}
		projectDir, _ := filepath.Abs(e.Dir)
		projectName := filepath.Base(projectDir)
		e.TempDir = TempDir{
			Remote:      tempDir,
			Fingerprint: filepathext.SmartJoin(tempDir, projectName),
		}

	} else {
		e.TempDir = TempDir{
			Remote:      filepathext.SmartJoin(e.Dir, tempDir),
			Fingerprint: filepathext.SmartJoin(e.Dir, tempDir),
		}
	}

	// RemoteCacheDir from taskrc/env can override the remote cache directory
	if e.RemoteCacheDir != "" {
		if filepath.IsAbs(e.RemoteCacheDir) || strings.HasPrefix(e.RemoteCacheDir, "~") {
			remoteCacheDir, err := execext.ExpandLiteral(e.RemoteCacheDir)
			if err != nil {
				return err
			}
			e.TempDir.Remote = remoteCacheDir
		} else {
			e.TempDir.Remote = filepathext.SmartJoin(e.Dir, e.RemoteCacheDir)
		}
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
	e.Output, err = output.BuildFor(&e.OutputStyle, e.Logger)
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

	e.Compiler = &Compiler{
		Dir:            e.Dir,
		Entrypoint:     e.Entrypoint,
		UserWorkingDir: e.UserWorkingDir,
		TaskfileEnv:    e.Taskfile.Env,
		TaskfileVars:   e.Taskfile.Vars,
		Logger:         e.Logger,
	}
	return nil
}

func (e *Executor) readDotEnvFiles() error {
	if e.Taskfile == nil || len(e.Taskfile.Dotenv) == 0 {
		return nil
	}

	if e.Taskfile.Version.LessThan(ast.V3) {
		return nil
	}

	vars, err := e.Compiler.GetTaskfileVariables()
	if err != nil {
		return err
	}

	env, err := taskfile.Dotenv(vars, e.Taskfile, e.Dir)
	if err != nil {
		return err
	}

	for k, v := range env.All() {
		if _, ok := e.Taskfile.Env.Get(k); !ok {
			e.Taskfile.Env.Set(k, v)
		}
	}
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
	for k := range e.Taskfile.Tasks.Keys(nil) {
		e.taskCallCount[k] = new(int32)
		e.mkdirMutexMap[k] = &sync.Mutex{}
	}

	if e.Concurrency > 0 {
		e.concurrencySemaphore = make(chan struct{}, e.Concurrency)
	}
}

func (e *Executor) doVersionChecks() error {
	if !e.EnableVersionCheck {
		return nil
	}
	// Copy the version to avoid modifying the original
	schemaVersion := &semver.Version{}
	*schemaVersion = *e.Taskfile.Version

	// Error if the Taskfile uses a schema version below v3
	if schemaVersion.LessThan(ast.V3) {
		return &errors.TaskfileVersionCheckError{
			URI:           e.Taskfile.Location,
			SchemaVersion: schemaVersion,
			Message:       `no longer supported. Please use v3 or above`,
		}
	}

	// Get the current version of Task
	// If we can't parse the version (e.g. when its "devel"), then ignore the current version checks
	currentVersion, err := semver.NewVersion(version.GetVersion())
	if err != nil {
		return nil
	}

	// Error if the Taskfile uses a schema version above the current version of Task
	if schemaVersion.GreaterThan(currentVersion) {
		return &errors.TaskfileVersionCheckError{
			URI:           e.Taskfile.Location,
			SchemaVersion: schemaVersion,
			Message:       fmt.Sprintf(`is greater than the current version of Task (%s)`, currentVersion.String()),
		}
	}

	return nil
}
