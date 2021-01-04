package task

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
	"github.com/mattn/go-zglob"
	"github.com/radovskyb/watcher"
)

var watchIgnoredDirs = []string{
	".git",
	"node_modules",
}

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

	w := watcher.New()
	defer w.Close()
	w.SetMaxEvents(1)
	if err := w.Ignore(watchIgnoredDirs...); err != nil {
		cancel()
		return err
	}

	closeOnInterrupt(w)

	go func() {
		for {
			select {
			case event := <-w.Event:
				e.Logger.VerboseErrf(logger.Magenta, "task: received watch event: %v", event)

				cancel()
				ctx, cancel = context.WithCancel(context.Background())
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
					go func() {
						w.TriggerEvent(watcher.Remove, nil)
					}()
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
		// re-register every 20 seconds because we can have new files, but this process is expensive to run
		for {
			if err := e.registerWatchedFiles(w, calls...); err != nil {
				e.Logger.Errf(logger.Red, "%v", err)
			}
			time.Sleep(time.Second * 20)
		}
	}()

	return w.Start(time.Second)
}

func isContextError(err error) bool {
	if taskRunErr, ok := err.(*taskRunError); ok {
		err = taskRunErr.err
	}

	return err == context.Canceled || err == context.DeadlineExceeded
}

func closeOnInterrupt(w *watcher.Watcher) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		<-ch
		w.Close()
	}()
}

func (e *Executor) registerWatchedFiles(w *watcher.Watcher, calls ...taskfile.Call) error {
	oldWatchedFiles := make(map[string]struct{})
	for f := range w.WatchedFiles() {
		oldWatchedFiles[f] = struct{}{}
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
			files, err := zglob.Glob(s)
			if err != nil {
				return err
			}
			for _, f := range files {
				absFile, err := filepath.Abs(f)
				if err != nil {
					return err
				}
				if _, ok := oldWatchedFiles[absFile]; ok {
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
