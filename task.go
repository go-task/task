package task

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-task/task/execext"

	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
)

var (
	// TaskFilePath is the default Taskfile
	TaskFilePath = "Taskfile"

	// Force (--force or -f flag) forces a task to run even when it's up-to-date
	Force bool
	// Watch (--watch or -w flag) enables watch of a task
	Watch bool

	// Tasks constains the tasks parsed from Taskfile
	Tasks = make(map[string]*Task)
)

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
func Run() {
	log.SetFlags(0)

	args := pflag.Args()
	if len(args) == 0 {
		log.Println("task: No argument given, trying default task")
		args = []string{"default"}
	}

	var err error
	Tasks, err = readTaskfile()
	if err != nil {
		log.Fatal(err)
	}

	if HasCyclicDep(Tasks) {
		log.Fatal("Cyclic dependency detected")
	}

	// check if given tasks exist
	for _, a := range args {
		if _, ok := Tasks[a]; !ok {
			var err error = &taskNotFoundError{taskName: a}
			fmt.Println(err)
			printExistingTasksHelp()
			return
		}
	}

	if Watch {
		if err := WatchTasks(args); err != nil {
			log.Fatal(err)
		}
		return
	}

	for _, a := range args {
		if err = RunTask(context.Background(), a); err != nil {
			log.Fatal(err)
		}
	}
}

// RunTask runs a task by its name
func RunTask(ctx context.Context, name string) error {
	t, ok := Tasks[name]
	if !ok {
		return &taskNotFoundError{name}
	}

	if err := t.runDeps(ctx); err != nil {
		return err
	}

	if !Force {
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
		if err := t.runCommand(ctx, i); err != nil {
			return &taskRunError{name, err}
		}
	}
	return nil
}

func (t *Task) runDeps(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, d := range t.Deps {
		dep := d

		g.Go(func() error {
			dep, err := t.ReplaceVariables(dep)
			if err != nil {
				return err
			}

			if err = RunTask(ctx, dep); err != nil {
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

func (t *Task) runCommand(ctx context.Context, i int) error {
	c, err := t.ReplaceVariables(t.Cmds[i])
	if err != nil {
		return err
	}

	if strings.HasPrefix(c, "^") {
		c = strings.TrimPrefix(c, "^")
		if err = RunTask(ctx, c); err != nil {
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
