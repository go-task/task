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

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/status"
	"github.com/go-task/task/v3/taskfile"
	"github.com/radovskyb/watcher"
)

const defaultWatchInterval = 5 * time.Second

// watchTasks start watching the given tasks
func (e *Executor) watchTasks(calls ...taskfile.Call) error {
	tasks := make([]string, len(calls))
	for i, c := range calls {
		tasks[i] = c.Task
	}

	e.Logger.Errf(logger.Green, "task: Started watching for tasks: %s", strings.Join(tasks, ", "))

	ctx, cancel := context.WithCancel(context.Background())
	for _, c := range calls {
		c := c
		go func() {
			if err := e.RunTask(ctx, c); err != nil && !isContextError(err) {
				e.Logger.Errf(logger.Red, "%v", err)
			}
		}()
	}

	var watchIntervalString string

	if e.Interval != "" {
		watchIntervalString = e.Interval
	} else if e.Taskfile.Interval != "" {
		watchIntervalString = e.Taskfile.Interval
	}

	watchInterval := defaultWatchInterval

	if watchIntervalString != "" {
		var err error
		watchInterval, err = parseWatchInterval(watchIntervalString)
		if err != nil {
			cancel()
			return err
		}
	}

	e.Logger.VerboseOutf(logger.Green, "task: Watching for changes every %v", watchInterval)

	w := watcher.New()
	defer w.Close()
	w.SetMaxEvents(1)

	closeOnInterrupt(w)

	go func() {
		for {
			select {
			case event := <-w.Event:
				e.Logger.VerboseErrf(logger.Magenta, "task: received watch event: %v", event)

				cancel()
				ctx, cancel = context.WithCancel(context.Background())

				e.Compiler.ResetCache()

				for _, c := range calls {
					c := c
					go func() {
						if err := e.RunTask(ctx, c); err != nil && !isContextError(err) {
							e.Logger.Errf(logger.Red, "%v", err)
						}
					}()
				}
			case err := <-w.Error:
				switch err {
				case watcher.ErrWatchedFileDeleted:
				default:
					e.Logger.Errf(logger.Red, "%v", err)
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
				e.Logger.Errf(logger.Red, "%v", err)
			}
			time.Sleep(watchInterval)
		}
	}()

	return w.Start(watchInterval)
}

func isContextError(err error) bool {
	if taskRunErr, ok := err.(*TaskRunError); ok {
		err = taskRunErr.err
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

		for _, s := range task.Sources {
			files, err := status.Glob(task.Dir, s)
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
				e.Logger.VerboseOutf(logger.Green, "task: watching new file: %v", absFile)
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
	return strings.Contains(path, "/.git") || strings.Contains(path, "/.task") || strings.Contains(path, "/node_modules")
}

func parseWatchInterval(watchInterval string) (time.Duration, error) {
	v, err := time.ParseDuration(watchInterval)
	if err != nil {
		return 0, fmt.Errorf(`task: Could not parse watch interval "%s": %v`, watchInterval, err)
	}
	return v, nil
}
