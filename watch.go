package task

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

// watchTasks start watching the given tasks
func (e *Executor) watchTasks(calls ...*ast.Call) error {
	tasks := make([]string, len(calls))
	for i, c := range calls {
		tasks[i] = c.Task
	}

	e.Logger.Errf(logger.Green, "task: Started watching for tasks: %s\n", strings.Join(tasks, ", "))

	ctx, cancel := context.WithCancel(context.Background())
	for _, c := range calls {
		c := c
		go func() {
			err := e.RunTask(ctx, c)
			if err == nil {
				e.Logger.Errf(logger.Green, "task: task \"%s\" finished running\n", c.Task)
			} else if !isContextError(err) {
				e.Logger.Errf(logger.Red, "%v\n", err)
			}
		}()
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		cancel()
		return err
	}
	defer w.Close()

	closeOnInterrupt(w)

	go func() {
		for {
			select {
			case event, ok := <-w.Events:
				switch {
				case !ok:
					cancel()
					return
				case event.Op == fsnotify.Chmod:
					continue
				}
				e.Logger.VerboseErrf(logger.Magenta, "task: received watch event: %v\n", event)

				e.Compiler.ResetCache()

				for _, c := range calls {
					c := c
					go func() {
						t, err := e.GetTask(c)
						if err != nil {
							e.Logger.Errf(logger.Red, "%v\n", err)
							return
						}
						files, err := fingerprint.Globs(t.Dir, t.Sources)
						if err != nil {
							e.Logger.Errf(logger.Red, "%v\n", err)
							return
						}
						relPath, _ := filepath.Rel(e.Dir, event.Name)
						if !slices.Contains(files, relPath) {
							e.Logger.VerboseErrf(logger.Magenta, "task: skipped for file not in sources: %s\n", relPath)
							return
						}
						err = e.RunTask(ctx, c)
						if err == nil {
							e.Logger.Errf(logger.Green, "task: task \"%s\" finished running\n", c.Task)
						} else if !isContextError(err) {
							e.Logger.Errf(logger.Red, "%v\n", err)
						}
					}()
				}
			case err, ok := <-w.Errors:
				switch {
				case !ok:
					cancel()
					return
				default:
					e.Logger.Errf(logger.Red, "%v\n", err)
				}
			}
		}
	}()

	if err := e.registerWatchedDirs(w, calls...); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func isContextError(err error) bool {
	if taskRunErr, ok := err.(*errors.TaskRunError); ok {
		err = taskRunErr.Err
	}

	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func closeOnInterrupt(w *fsnotify.Watcher) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		w.Close()
	}()
}

func (e *Executor) registerWatchedDirs(w *fsnotify.Watcher, calls ...*ast.Call) error {
	watchedDirs := make(map[string]bool)

	var registerTaskDirs func(*ast.Call) error
	registerTaskDirs = func(c *ast.Call) error {
		task, err := e.CompiledTask(c)
		if err != nil {
			return err
		}

		for _, d := range task.Deps {
			if err := registerTaskDirs(&ast.Call{Task: d.Task, Vars: d.Vars}); err != nil {
				return err
			}
		}
		for _, c := range task.Cmds {
			if c.Task != "" {
				if err := registerTaskDirs(&ast.Call{Task: c.Task, Vars: c.Vars}); err != nil {
					return err
				}
			}
		}

		dirs, err := fingerprint.GlobsDirs(task.Dir, task.Sources)
		if err != nil {
			return err
		}

		for _, d := range dirs {
			if watchedDirs[d] {
				continue
			}
			if ShouldIgnoreFile(d) {
				continue
			}
			if err := w.Add(d); err != nil {
				return err
			}
			watchedDirs[d] = true
			relPath, _ := filepath.Rel(e.Dir, d)
			e.Logger.VerboseOutf(logger.Green, "task: watching new dir: %v\n", relPath)
		}
		return nil
	}

	for _, c := range calls {
		if err := registerTaskDirs(c); err != nil {
			return err
		}
	}
	return nil
}

func ShouldIgnoreFile(path string) bool {
	ignorePaths := []string{
		"/.task",
		"/.git",
		"/.hg",
		"/node_modules",
	}
	for _, p := range ignorePaths {
		if strings.Contains(path, fmt.Sprintf("%s/", p)) || strings.HasSuffix(path, p) {
			return true
		}
	}
	return false
}
