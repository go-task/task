package task

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-task/task/execext"

	"golang.org/x/sync/errgroup"
)

const (
	// TaskFilePath is the default Taskfile
	TaskFilePath = "Taskfile"
)

// Executor executes a Taskfile
type Executor struct {
	Tasks Tasks
	Force bool
	Watch bool

	watchingFiles map[string]struct{}
}

// Tasks representas a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Cmds      []string
	Deps      []string
	Desc      string
	Sources   []string
	Generates []string
	Status    []string
	Dir       string
	Vars      map[string]string
	Set       string
	Env       map[string]string
}

// Run runs Task
func (e *Executor) Run(args ...string) error {
	if e.HasCyclicDep() {
		return ErrCyclicDependencyDetected
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
		if err := e.RunTask(context.Background(), a); err != nil {
			return err
		}
	}
	return nil
}

// RunTask runs a task by its name
func (e *Executor) RunTask(ctx context.Context, name string) error {
	t, ok := e.Tasks[name]
	if !ok {
		return &taskNotFoundError{name}
	}

	if err := e.runDeps(ctx, name); err != nil {
		return err
	}

	if !e.Force {
		upToDate, err := t.isUpToDate(ctx)
		if err != nil {
			return err
		}
		if upToDate {
			log.Printf(`task: Task "%s" is up to date`, name)
			return nil
		}
	}

	for i := range t.Cmds {
		if err := e.runCommand(ctx, name, i); err != nil {
			return &taskRunError{name, err}
		}
	}
	return nil
}

func (e *Executor) runDeps(ctx context.Context, task string) error {
	g, ctx := errgroup.WithContext(ctx)
	t := e.Tasks[task]

	for _, d := range t.Deps {
		dep := d

		g.Go(func() error {
			dep, err := t.ReplaceVariables(dep)
			if err != nil {
				return err
			}

			if err = e.RunTask(ctx, dep); err != nil {
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

func (t *Task) isUpToDate(ctx context.Context) (bool, error) {
	if len(t.Status) > 0 {
		environ, err := t.getEnviron()
		if err != nil {
			return false, err
		}

		for _, s := range t.Status {
			err = execext.RunCommand(&execext.RunCommandOptions{
				Context: ctx,
				Command: s,
				Dir:     t.Dir,
				Env:     environ,
			})
			if err != nil {
				return false, nil
			}
		}
		return true, nil
	}

	if len(t.Sources) == 0 || len(t.Generates) == 0 {
		return false, nil
	}

	sources, err := t.ReplaceSliceVariables(t.Sources)
	if err != nil {
		return false, err
	}
	generates, err := t.ReplaceSliceVariables(t.Generates)
	if err != nil {
		return false, err
	}

	sourcesMaxTime, err := getPatternsMaxTime(sources)
	if err != nil || sourcesMaxTime.IsZero() {
		return false, nil
	}

	generatesMinTime, err := getPatternsMinTime(generates)
	if err != nil || generatesMinTime.IsZero() {
		return false, nil
	}

	return generatesMinTime.After(sourcesMaxTime), nil
}

func (e *Executor) runCommand(ctx context.Context, task string, i int) error {
	t := e.Tasks[task]

	c, err := t.ReplaceVariables(t.Cmds[i])
	if err != nil {
		return err
	}

	if strings.HasPrefix(c, "^") {
		c = strings.TrimPrefix(c, "^")
		if err = e.RunTask(ctx, c); err != nil {
			return err
		}
		return nil
	}

	dir, err := t.ReplaceVariables(t.Dir)
	if err != nil {
		return err
	}

	envs, err := t.getEnviron()
	if err != nil {
		return err
	}
	opts := &execext.RunCommandOptions{
		Context: ctx,
		Command: c,
		Dir:     dir,
		Env:     envs,
		Stdin:   os.Stdin,
		Stderr:  os.Stderr,
	}

	if t.Set == "" {
		log.Println(c)
		opts.Stdout = os.Stdout
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

func (t *Task) getEnviron() ([]string, error) {
	if t.Env == nil {
		return nil, nil
	}

	envs := os.Environ()

	for k, v := range t.Env {
		env, err := t.ReplaceVariables(fmt.Sprintf("%s=%s", k, v))
		if err != nil {
			return nil, err
		}
		envs = append(envs, env)
	}
	return envs, nil
}
