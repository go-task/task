package task

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mattn/go-zglob"
)

// WatchTasks start watching the given tasks
func WatchTasks(args []string) error {
	log.Printf("task: Started watching for tasks: %s", strings.Join(args, ", "))

	// run tasks on init
	for _, a := range args {
		if err := RunTask(context.Background(), a); err != nil {
			fmt.Println(err)
			break
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	go func() {
		for {
			if err := registerWatchedFiles(watcher, args); err != nil {
				log.Printf("Error watching files: %v", err)
			}
			time.Sleep(time.Second * 2)
		}
	}()

loop:
	for {
		select {
		case <-watcher.Events:
			for _, a := range args {
				if err := RunTask(context.Background(), a); err != nil {
					fmt.Println(err)
					continue loop
				}
			}
		case err := <-watcher.Errors:
			fmt.Println(err)
			continue loop
		}
	}
}

var watchingFiles map[string]struct{}

func registerWatchedFiles(w *fsnotify.Watcher, args []string) error {
	oldWatchingFiles := watchingFiles
	watchingFiles = make(map[string]struct{}, len(oldWatchingFiles))

	for k := range oldWatchingFiles {
		if err := w.Remove(k); err != nil {
			return err
		}
	}

	for _, a := range args {
		task, ok := Tasks[a]
		if !ok {
			return &taskNotFoundError{a}
		}
		if err := registerWatchedFiles(w, task.Deps); err != nil {
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
				watchingFiles[f] = struct{}{}

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
