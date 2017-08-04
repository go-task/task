package task

import (
	"context"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mattn/go-zglob"
)

// watchTasks start watching the given tasks
func (e *Executor) watchTasks(args ...string) error {
	e.printfln("task: Started watching for tasks: %s", strings.Join(args, ", "))

	var isCtxErr = func(err error) bool {
		switch err {
		case context.Canceled, context.DeadlineExceeded:
			return true
		}
		return false
	}

	ctx, cancel := context.WithCancel(context.Background())
	for _, a := range args {
		a := a
		go func() {
			if err := e.RunTask(ctx, Call{Task: a}); err != nil && !isCtxErr(err) {
				e.println(err)
			}
		}()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	go func() {
		for {
			if err := e.registerWatchedFiles(watcher, args); err != nil {
				e.printfln("Error watching files: %v", err)
			}
			time.Sleep(time.Second * 2)
		}
	}()

	for {
		select {
		case <-watcher.Events:
			cancel()
			ctx, cancel = context.WithCancel(context.Background())
			for _, a := range args {
				a := a
				go func() {
					if err := e.RunTask(ctx, Call{Task: a}); err != nil && !isCtxErr(err) {
						e.println(err)
					}
				}()
			}
		case err := <-watcher.Errors:
			e.println(err)
		}
	}
}

func (e *Executor) registerWatchedFiles(w *fsnotify.Watcher, args []string) error {
	oldWatchingFiles := e.watchingFiles
	e.watchingFiles = make(map[string]struct{}, len(oldWatchingFiles))

	for k := range oldWatchingFiles {
		if err := w.Remove(k); err != nil {
			return err
		}
	}

	for _, a := range args {
		task, ok := e.Tasks[a]
		if !ok {
			return &taskNotFoundError{a}
		}
		deps := make([]string, len(task.Deps))
		for i, d := range task.Deps {
			deps[i] = d.Task
		}
		if err := e.registerWatchedFiles(w, deps); err != nil {
			return err
		}
		for _, s := range task.Sources {
			files, err := zglob.Glob(s)
			if err != nil {
				return err
			}
			for _, f := range files {
				if err := w.Add(f); err != nil {
					return err
				}
				e.watchingFiles[f] = struct{}{}

				// run if is new file
				if oldWatchingFiles != nil {
					if _, ok := oldWatchingFiles[f]; !ok {
						w.Events <- fsnotify.Event{Name: f, Op: fsnotify.Create}
					}
				}
			}
		}
	}
	return nil
}
