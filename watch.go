package task

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/radovskyb/watcher"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
)

const defaultWatchInterval = 5 * time.Second

// watchTasks start watching the given tasks
func (e *Executor) watchTasks(calls ...taskfile.Call) error {
	tasks := make([]string, len(calls))
	for i, c := range calls {
		tasks[i] = c.Task
	}

	e.Logger.Errf(logger.Green, "task: Started watching for tasks: %s\n", strings.Join(tasks, ", "))

	ctx, cancel := context.WithCancel(context.Background())
	for _, c := range calls {
		c := c
		go func() {
			if err := e.RunTask(ctx, c); err != nil && !isContextError(err) {
				e.Logger.Errf(logger.Red, "%v\n", err)
			}
		}()
	}

	var watchInterval time.Duration
	switch {
	case e.Interval != 0:
		watchInterval = e.Interval
	case e.Taskfile.Interval != 0:
		watchInterval = e.Taskfile.Interval
	default:
		watchInterval = defaultWatchInterval
	}

	e.Logger.VerboseOutf(logger.Green, "task: Watching for changes every %v\n", watchInterval)

	w := watcher.New()
	defer w.Close()
	w.SetMaxEvents(1)

	closeOnInterrupt(w)

	go func() {
		for {
			select {
			case event := <-w.Event:
				e.Logger.VerboseErrf(logger.Magenta, "task: received watch event: %v\n", event)

				cancel()
				ctx, cancel = context.WithCancel(context.Background())

				e.Compiler.ResetCache()

				for _, c := range calls {
					c := c
					go func() {
						if err := e.RunTask(ctx, c); err != nil && !isContextError(err) {
							e.Logger.Errf(logger.Red, "%v\n", err)
						}
					}()
				}
			case err := <-w.Error:
				switch err {
				case watcher.ErrWatchedFileDeleted:
				default:
					e.Logger.Errf(logger.Red, "%v\n", err)
				}
			case <-w.Closed:
				cancel()
				return
			}
		}
	}()

	go func() {
		// re-register every 5 seconds because we can have new files, but this process is expensive to run
		for {
			if err := e.registerWatchedFiles(w, calls...); err != nil {
				e.Logger.Errf(logger.Red, "%v\n", err)
			}
			time.Sleep(watchInterval)
		}
	}()

	return w.Start(watchInterval)
}

func isContextError(err error) bool {
	if taskRunErr, ok := err.(*errors.TaskRunError); ok {
		err = taskRunErr.Err
	}

	return err == context.Canceled || err == context.DeadlineExceeded
}

func closeOnInterrupt(w *watcher.Watcher) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		w.Close()
	}()
}

func (e *Executor) registerWatchedFiles(w *watcher.Watcher, calls ...taskfile.Call) error {
	watchedFiles := w.WatchedFiles()

	var registerTaskFiles func(taskfile.Call) error
	registerTaskFiles = func(c taskfile.Call) error {
		task, err := e.CompiledTask(c)
		if err != nil {
			return err
		}

		for _, d := range task.Deps {
			if err := registerTaskFiles(taskfile.Call{Task: d.Task, Vars: d.Vars}); err != nil {
				return err
			}
		}
		for _, c := range task.Cmds {
			if c.Task != "" {
				if err := registerTaskFiles(taskfile.Call{Task: c.Task, Vars: c.Vars}); err != nil {
					return err
				}
			}
		}

		globs, err := fingerprint.Globs(task.Dir, task.Sources)
		if err != nil {
			return err
		}

		for _, s := range globs {
			files, err := fingerprint.Glob(task.Dir, s)
			if err != nil {
				return fmt.Errorf("task: %s: %w", s, err)
			}
			for _, f := range files {
				absFile, err := filepath.Abs(f)
				if err != nil {
					return err
				}
				if shouldIgnoreFile(absFile) {
					continue
				}
				if _, ok := watchedFiles[absFile]; ok {
					continue
				}
				if err := w.Add(absFile); err != nil {
					return err
				}
				e.Logger.VerboseOutf(logger.Green, "task: watching new file: %v\n", absFile)
			}
		}
		return nil
	}

	for _, c := range calls {
		if err := registerTaskFiles(c); err != nil {
			return err
		}
	}
	return nil
}

func shouldIgnoreFile(path string) bool {
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
