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

	"github.com/fsnotify/fsnotify"

	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
)

const defaultWatchInterval = 5 * time.Second

// watchTasks start watching the given tasks
func (e *Executor) watchTasks(calls ...taskfile.Call) (bool, error) {
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

	var watchInterval time.Duration
	switch {
	case e.Interval != 0:
		watchInterval = e.Interval
	case e.Taskfile.Interval != 0:
		watchInterval = e.Taskfile.Interval
	default:
		watchInterval = defaultWatchInterval
	}

	e.Logger.VerboseOutf(logger.Green, "task: Watching for changes every %v", watchInterval)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return false, err
    }
	defer w.Close()

	keepRunning := make(chan bool)
	reload := false

	go func() {
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				e.Logger.VerboseErrf(logger.Magenta, "task: received watch event: %v", event)

				if event.Name == e.Taskfile.Location {
					e.Logger.VerboseErrf(logger.Magenta, "task: reload taskfile")
					reload = true
					keepRunning <- false
					return
				}

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
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				e.Logger.Errf(logger.Red, "%v", err)
			}
		}
	}()

	go func() {
		// re-register every 5 seconds because we can have new files, but this process is expensive to run
		for !reload {
			if err := e.registerWatchedFiles(w, calls...); err != nil {
				e.Logger.Errf(logger.Red, "%v", err)
			}
			time.Sleep(watchInterval)
		}
	}()

	e.Logger.VerboseOutf(logger.Green, "task: watching taskfile: %v", e.Taskfile.Location)
	if err := w.Add(e.Taskfile.Location); err != nil {
		e.Logger.Errf(logger.Red, "%v", err)
	}

	waitForInterrupt(keepRunning)
	return reload, nil
}

func isContextError(err error) bool {
	if taskRunErr, ok := err.(*TaskRunError); ok {
		err = taskRunErr.err
	}

	return err == context.Canceled || err == context.DeadlineExceeded
}

func waitForInterrupt(keepRunning chan bool) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		keepRunning <- false
	}()

	<-keepRunning
}

func (e *Executor) registerWatchedFiles(w *fsnotify.Watcher, calls ...taskfile.Call) error {
	watchedFiles := make(map[string]bool)
	for _, file := range w.WatchList() {
		watchedFiles[file] = true
	}

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
