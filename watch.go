package task

import (
	"context"
	"strings"
	"time"

	"github.com/go-task/task/internal/taskfile"
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
	e.errf("task: Started watching for tasks: %s", strings.Join(tasks, ", "))

	ctx, cancel := context.WithCancel(context.Background())
	for _, c := range calls {
		c := c
		go func() {
			if err := e.RunTask(ctx, c); err != nil && !isContextError(err) {
				e.errf("%v", err)
			}
		}()
	}

	w := watcher.New()
	defer w.Close()
	w.SetMaxEvents(1)
	if err := w.Ignore(watchIgnoredDirs...); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event := <-w.Event:
				e.verboseErrf("task: received watch event: %v", event)

				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				for _, c := range calls {
					c := c
					go func() {
						if err := e.RunTask(ctx, c); err != nil && !isContextError(err) {
							e.errf("%v", err)
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
					e.errf("%v", err)
				}
			case <-w.Closed:
				return
			}
		}
	}()

	go func() {
		// re-register each second because we can have new files
		for {
			if err := e.registerWatchedFiles(w, calls...); err != nil {
				e.errf("%v", err)
			}
			time.Sleep(time.Second)
		}
	}()

	return w.Start(time.Second)
}

func (e *Executor) registerWatchedFiles(w *watcher.Watcher, calls ...taskfile.Call) error {
	oldWatchedFiles := make(map[string]struct{})
	for f := range w.WatchedFiles() {
		oldWatchedFiles[f] = struct{}{}
	}

	for f := range oldWatchedFiles {
		if err := w.Remove(f); err != nil {
			return err
		}
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
				if _, ok := oldWatchedFiles[f]; ok {
					continue
				}
				if err := w.Add(f); err != nil {
					return err
				}
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

func isContextError(err error) bool {
	switch err {
	case context.Canceled, context.DeadlineExceeded:
		return true
	default:
		return false
	}
}
