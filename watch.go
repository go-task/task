package task

import (
	"context"
	"strings"
	"time"

	"github.com/mattn/go-zglob"
	"github.com/radovskyb/watcher"
)

var watchIgnoredDirs = []string{
	".git",
	"node_modules",
}

// watchTasks start watching the given tasks
func (e *Executor) watchTasks(args ...string) error {
	e.printfln("task: Started watching for tasks: %s", strings.Join(args, ", "))

	ctx, cancel := context.WithCancel(context.Background())
	for _, a := range args {
		a := a
		go func() {
			if err := e.RunTask(ctx, Call{Task: a}); err != nil && !isContextError(err) {
				e.println(err)
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
				e.verbosePrintfln("task: received watch event: %v", event)

				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				for _, a := range args {
					a := a
					go func() {
						if err := e.RunTask(ctx, Call{Task: a}); err != nil && !isContextError(err) {
							e.println(err)
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
					e.println(err)
				}
			case <-w.Closed:
				return
			}
		}
	}()

	go func() {
		// re-register each second because we can have new files
		for {
			if err := e.registerWatchedFiles(w, args); err != nil {
				e.println(err)
			}
			time.Sleep(time.Second)
		}
	}()

	return w.Start(time.Second)
}

func (e *Executor) registerWatchedFiles(w *watcher.Watcher, args []string) error {
	oldWatchedFiles := make(map[string]struct{})
	for f := range w.WatchedFiles() {
		oldWatchedFiles[f] = struct{}{}
	}

	for f := range oldWatchedFiles {
		if err := w.Remove(f); err != nil {
			return err
		}
	}

	var registerTaskFiles func(string) error
	registerTaskFiles = func(t string) error {
		task, ok := e.Tasks[t]
		if !ok {
			return &taskNotFoundError{t}
		}

		for _, d := range task.Deps {
			if err := registerTaskFiles(d.Task); err != nil {
				return err
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

	for _, a := range args {
		if err := registerTaskFiles(a); err != nil {
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
