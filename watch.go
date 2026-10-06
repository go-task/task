package task

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/helshabini/fsbroker"
	"github.com/puzpuzpuz/xsync/v4"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/fingerprint"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/slicesext"
	"github.com/go-task/task/v3/taskfile/ast"
)

const defaultWaitTime = 100 * time.Millisecond

// watchTasks start watching the given tasks
func (e *Executor) watchTasks(calls ...*Call) error {
	tasks := make([]string, len(calls))
	for i, c := range calls {
		tasks[i] = c.Task
	}

	e.Logger.Errf(logger.Green, "task: Started watching for tasks: %s\n", strings.Join(tasks, ", "))

	ctx, cancel := context.WithCancel(context.Background())
	for _, c := range calls {
		go func() {
			err := e.RunTask(ctx, c)
			if err == nil {
				e.Logger.Errf(logger.Green, "task: task \"%s\" finished running\n", c.Task)
			} else if !isContextError(err) {
				e.Logger.Errf(logger.Red, "%v\n", err)
			}
		}()
	}

	var waitTime time.Duration
	switch {
	case e.Interval != 0:
		waitTime = e.Interval
	case e.Taskfile.Interval != 0:
		waitTime = e.Taskfile.Interval
	default:
		waitTime = defaultWaitTime
	}

	config := fsbroker.DefaultFSConfig()
	config.Timeout = waitTime
	config.IgnoreHiddenFiles = false

	broker, err := fsbroker.NewFSBroker(config)
	if err != nil {
		cancel()
		return err
	}
	defer broker.Stop()

	closeOnInterrupt(broker)

	e.watchedDirs = xsync.NewMap[string, bool]()

	broker.Start()

	go func() {
		for {
			select {
			case action, ok := <-broker.Next():
				if !ok {
					cancel()
					return
				}

				actions := e.filterWatchActions(broker, drainActions(broker, action))
				if len(actions) == 0 {
					continue
				}

				files, err := e.collectSources(calls)
				if err != nil {
					e.Logger.Errf(logger.Red, "%v\n", err)
					continue
				}

				if !slices.ContainsFunc(actions, func(a *fsbroker.FSAction) bool {
					return isRelevantWatchAction(a, files)
				}) {
					for _, a := range actions {
						relPath, _ := filepath.Rel(e.Dir, a.Subject.Path)
						e.Logger.VerboseErrf(logger.Magenta, "task: skipped for file not in sources: %s\n", relPath)
					}
					continue
				}

				cancel()
				ctx, cancel = context.WithCancel(context.Background())

				e.Compiler.ResetCache()

				for _, c := range calls {
					go func(ctx context.Context, c *Call) {
						err := e.RunTask(ctx, c)
						if err == nil {
							e.Logger.Errf(logger.Green, "task: task \"%s\" finished running\n", c.Task)
						} else if !isContextError(err) {
							e.Logger.Errf(logger.Red, "%v\n", err)
						}
					}(ctx, c)
				}
			case err, ok := <-broker.Error():
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

	go func() {
		// NOTE(@andreynering): New files can be created in directories
		// that were previously empty, so we need to check for new dirs
		// from time to time.
		for {
			if err := e.registerWatchedDirs(broker, calls...); err != nil {
				e.Logger.Errf(logger.Red, "%v\n", err)
			}
			time.Sleep(5 * time.Second)
		}
	}()

	<-make(chan struct{})
	return nil
}

func isContextError(err error) bool {
	if taskRunErr, ok := err.(*errors.TaskRunError); ok {
		err = taskRunErr.Err
	}

	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func closeOnInterrupt(broker *fsbroker.FSBroker) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		broker.Stop()
		os.Exit(0)
	}()
}

// drainActions returns the given action along with any other action that is
// already available, so that a burst of changes (e.g. a git checkout) results
// in a single task run instead of one per changed file.
func drainActions(broker *fsbroker.FSBroker, action *fsbroker.FSAction) []*fsbroker.FSAction {
	actions := []*fsbroker.FSAction{action}
	for {
		select {
		case next, ok := <-broker.Next():
			if !ok {
				return actions
			}
			actions = append(actions, next)
		default:
			return actions
		}
	}
}

// filterWatchActions applies the side effects of the given actions and returns
// the ones that are candidates for triggering a task run.
func (e *Executor) filterWatchActions(broker *fsbroker.FSBroker, actions []*fsbroker.FSAction) []*fsbroker.FSAction {
	filtered := make([]*fsbroker.FSAction, 0, len(actions))
	for _, action := range actions {
		if action.Subject == nil || action.Type == fsbroker.NoOp || action.Type == fsbroker.Chmod {
			continue
		}

		e.Logger.VerboseErrf(logger.Magenta, "task: received watch event: %s: %s\n", action.Type, action.Subject.Path)

		if action.Type == fsbroker.Remove || action.Type == fsbroker.Rename {
			e.watchedDirs.Delete(action.Subject.Path)
			if oldPath, ok := action.Properties["OldPath"].(string); ok {
				e.watchedDirs.Delete(oldPath)
			}
		}

		if ShouldIgnore(action.Subject.Path) {
			e.Logger.VerboseErrf(logger.Magenta, "task: event skipped for being an ignored dir: %s\n", action.Subject.Path)
			continue
		}

		if action.Subject.IsDir() && action.Type != fsbroker.Remove {
			if err := e.registerWatchedTree(broker, action.Subject.Path); err != nil {
				e.Logger.VerboseErrf(logger.Magenta, "task: failed to watch dir %s: %v\n", action.Subject.Path, err)
			}
		}

		filtered = append(filtered, action)
	}
	return filtered
}

func isRelevantWatchAction(action *fsbroker.FSAction, files []string) bool {
	if action.Subject.IsDir() || action.Type == fsbroker.Remove || action.Type == fsbroker.Rename {
		return true
	}
	return slices.Contains(files, filepath.ToSlash(action.Subject.Path))
}

func (e *Executor) registerWatchedDirs(broker *fsbroker.FSBroker, calls ...*Call) error {
	files, err := e.collectSources(calls)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := e.registerWatchedDir(broker, filepath.Dir(f)); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) registerWatchedTree(broker *fsbroker.FSBroker, dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			e.Logger.VerboseErrf(logger.Magenta, "task: failed to walk dir %s: %v\n", path, err)
			return filepath.SkipDir
		}
		if !d.IsDir() {
			return nil
		}
		if ShouldIgnore(path) {
			return filepath.SkipDir
		}
		return e.registerWatchedDir(broker, path)
	})
}

func (e *Executor) registerWatchedDir(broker *fsbroker.FSBroker, d string) error {
	if isSet, ok := e.watchedDirs.Load(d); ok && isSet {
		return nil
	}
	if ShouldIgnore(d) {
		return nil
	}
	if err := broker.AddWatch(d); err != nil {
		return err
	}
	e.watchedDirs.Store(d, true)
	relPath, _ := filepath.Rel(e.Dir, d)
	e.Logger.VerboseOutf(logger.Green, "task: watching new dir: %v\n", relPath)
	return nil
}

var ignorePaths = []string{
	"/.task",
	"/.git",
	"/.hg",
	"/node_modules",
}

func ShouldIgnore(path string) bool {
	for _, p := range ignorePaths {
		if strings.Contains(path, fmt.Sprintf("%s/", p)) || strings.HasSuffix(path, p) {
			return true
		}
	}
	return false
}

func (e *Executor) collectSources(calls []*Call) ([]string, error) {
	var sources []string

	err := e.traverse(calls, func(task *ast.Task) error {
		files, err := fingerprint.Globs(task.Dir, task.Sources, task.ShouldUseGitignore())
		if err != nil {
			return err
		}
		sources = append(sources, files...)
		return nil
	})

	return slicesext.UniqueJoin(sources), err
}

type traverseFunc func(*ast.Task) error

func (e *Executor) traverse(calls []*Call, yield traverseFunc) error {
	for _, c := range calls {
		task, err := e.CompiledTask(c)
		if err != nil {
			return err
		}
		for _, dep := range task.Deps {
			if dep.Task != "" {
				if err := e.traverse([]*Call{{Task: dep.Task, Vars: dep.Vars}}, yield); err != nil {
					return err
				}
			}
		}
		for _, cmd := range task.Cmds {
			if cmd.Task != "" {
				if err := e.traverse([]*Call{{Task: cmd.Task, Vars: cmd.Vars}}, yield); err != nil {
					return err
				}
			}
		}
		if err := yield(task); err != nil {
			return err
		}
	}
	return nil
}
