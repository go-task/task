package task

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-task/task/execext"

	"golang.org/x/sync/errgroup"
)

const (
	// TaskFilePath is the default Taskfile
	TaskFilePath = "Taskfile"
)

// Executor executes a Taskfile
type Executor struct {
	Tasks   Tasks
	Dir     string
	Force   bool
	Watch   bool
	Verbose bool

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	taskvars      Vars
	watchingFiles map[string]struct{}

	dynamicCache   Vars
	muDynamicCache sync.Mutex
}

// Vars is a string[string] variables map
type Vars map[string]string

// Tasks representas a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Cmds      []*Cmd
	Deps      []*Dep
	Desc      string
	Sources   []string
	Generates []string
	Status    []string
	Dir       string
	Vars      Vars
	Set       string
	Env       Vars
}

// Run runs Task
func (e *Executor) Run(args ...string) error {
	if e.HasCyclicDep() {
		return ErrCyclicDependencyDetected
	}

	if e.Stdin == nil {
		e.Stdin = os.Stdin
	}
	if e.Stdout == nil {
		e.Stdout = os.Stdout
	}
	if e.Stderr == nil {
		e.Stderr = os.Stderr
	}

	if e.dynamicCache == nil {
		e.dynamicCache = make(Vars, 10)
	}

	// check if given tasks exist
	for _, a := range args {
		if _, ok := e.Tasks[a]; !ok {
			// FIXME: move to the main package
			e.printExistingTasksHelp()
			return &taskNotFoundError{taskName: a}
		}
	}

	if e.Watch {
		if err := e.watchTasks(args...); err != nil {
			return err
		}
		return nil
	}

	for _, a := range args {
		if err := e.RunTask(context.Background(), a, nil); err != nil {
			return err
		}
	}
	return nil
}

// RunTask runs a task by its name
func (e *Executor) RunTask(ctx context.Context, name string, vars Vars) error {
	t, ok := e.Tasks[name]
	if !ok {
		return &taskNotFoundError{name}
	}

	if err := e.runDeps(ctx, name, vars); err != nil {
		return err
	}

	if !e.Force {
		upToDate, err := e.isTaskUpToDate(ctx, name, vars)
		if err != nil {
			return err
		}
		if upToDate {
			e.printfln(`task: Task "%s" is up to date`, name)
			return nil
		}
	}

	for i := range t.Cmds {
		if err := e.runCommand(ctx, name, i, vars); err != nil {
			return &taskRunError{name, err}
		}
	}
	return nil
}

func (e *Executor) runDeps(ctx context.Context, task string, vars Vars) error {
	g, ctx := errgroup.WithContext(ctx)
	t := e.Tasks[task]

	for _, d := range t.Deps {
		d := d

		g.Go(func() error {
			dep, err := e.ReplaceVariables(d.Task, task, vars)
			if err != nil {
				return err
			}

			if err = e.RunTask(ctx, dep, d.Vars); err != nil {
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func (e *Executor) isTaskUpToDate(ctx context.Context, task string, vars Vars) (bool, error) {
	t := e.Tasks[task]

	if len(t.Status) > 0 {
		return e.isUpToDateStatus(ctx, task, vars)
	}
	return e.isUpToDateTimestamp(ctx, task, vars)
}

func (e *Executor) isUpToDateStatus(ctx context.Context, task string, vars Vars) (bool, error) {
	t := e.Tasks[task]

	environ, err := e.getEnviron(task, vars)
	if err != nil {
		return false, err
	}
	dir, err := e.getTaskDir(task, vars)
	if err != nil {
		return false, err
	}
	status, err := e.ReplaceSliceVariables(t.Status, task, vars)
	if err != nil {
		return false, err
	}

	for _, s := range status {
		err = execext.RunCommand(&execext.RunCommandOptions{
			Context: ctx,
			Command: s,
			Dir:     dir,
			Env:     environ,
		})
		if err != nil {
			return false, nil
		}
	}
	return true, nil
}

func (e *Executor) isUpToDateTimestamp(ctx context.Context, task string, vars Vars) (bool, error) {
	t := e.Tasks[task]

	if len(t.Sources) == 0 || len(t.Generates) == 0 {
		return false, nil
	}

	dir, err := e.getTaskDir(task, vars)
	if err != nil {
		return false, err
	}

	sources, err := e.ReplaceSliceVariables(t.Sources, task, vars)
	if err != nil {
		return false, err
	}
	generates, err := e.ReplaceSliceVariables(t.Generates, task, vars)
	if err != nil {
		return false, err
	}

	sourcesMaxTime, err := getPatternsMaxTime(dir, sources)
	if err != nil || sourcesMaxTime.IsZero() {
		return false, nil
	}

	generatesMinTime, err := getPatternsMinTime(dir, generates)
	if err != nil || generatesMinTime.IsZero() {
		return false, nil
	}

	return generatesMinTime.After(sourcesMaxTime), nil
}

func (e *Executor) runCommand(ctx context.Context, task string, i int, vars Vars) error {
	t := e.Tasks[task]
	cmd := t.Cmds[i]

	if cmd.Cmd == "" {
		cmdVars := make(Vars, len(cmd.Vars))
		for k, v := range cmd.Vars {
			v, err := e.ReplaceVariables(v, task, vars)
			if err != nil {
				return err
			}
			cmdVars[k] = v
		}
		return e.RunTask(ctx, cmd.Task, cmdVars)
	}

	c, err := e.ReplaceVariables(cmd.Cmd, task, vars)
	if err != nil {
		return err
	}

	dir, err := e.getTaskDir(task, vars)
	if err != nil {
		return err
	}

	envs, err := e.getEnviron(task, vars)
	if err != nil {
		return err
	}
	opts := &execext.RunCommandOptions{
		Context: ctx,
		Command: c,
		Dir:     dir,
		Env:     envs,
		Stdin:   e.Stdin,
		Stderr:  e.Stderr,
	}

	e.println(c)
	if t.Set == "" {
		opts.Stdout = e.Stdout
		if err = execext.RunCommand(opts); err != nil {
			return err
		}
	} else {
		buff := bytes.NewBuffer(nil)
		opts.Stdout = buff
		if err = execext.RunCommand(opts); err != nil {
			return err
		}
		os.Setenv(t.Set, strings.TrimSpace(buff.String()))
	}
	return nil
}

func (e *Executor) getTaskDir(task string, vars Vars) (string, error) {
	t := e.Tasks[task]

	exeDir, err := e.ReplaceVariables(e.Dir, task, vars)
	if err != nil {
		return "", err
	}
	taskDir, err := e.ReplaceVariables(t.Dir, task, vars)
	if err != nil {
		return "", err
	}

	return filepath.Join(exeDir, taskDir), nil
}

func (e *Executor) getEnviron(task string, vars Vars) ([]string, error) {
	t := e.Tasks[task]

	if t.Env == nil {
		return nil, nil
	}

	envs := os.Environ()

	for k, v := range t.Env {
		env, err := e.ReplaceVariables(fmt.Sprintf("%s=%s", k, v), task, vars)
		if err != nil {
			return nil, err
		}
		envs = append(envs, env)
	}
	return envs, nil
}
