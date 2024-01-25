package taskfile

import (
	"context"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/filepathext"
	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/internal/templater"
	"github.com/go-task/task/v3/taskfile/ast"
)

const (
	taskfileUntrustedPrompt = `The task you are attempting to run depends on the remote Taskfile at %q.
--- Make sure you trust the source of this Taskfile before continuing ---
Continue?`
	taskfileChangedPrompt = `The Taskfile at %q has changed since you last used it!
--- Make sure you trust the source of this Taskfile before continuing ---
Continue?`
)

// Read reads a Read for a given directory
// Uses current dir when dir is left empty. Uses Read.yml
// or Read.yaml when entrypoint is left empty
func Read(
	node Node,
	insecure bool,
	download bool,
	offline bool,
	timeout time.Duration,
	tempDir string,
	l *logger.Logger,
) (*ast.Taskfile, error) {
	var _taskfile func(Node) (*ast.Taskfile, error)
	_taskfile = func(node Node) (*ast.Taskfile, error) {
		tf, err := readTaskfile(node, download, offline, timeout, tempDir, l)
		if err != nil {
			return nil, err
		}

		// Check that the Taskfile is set and has a schema version
		if tf == nil || tf.Version == nil {
			return nil, &errors.TaskfileVersionCheckError{URI: node.Location()}
		}

		if dir := node.BaseDir(); dir != "" {
			_ = tf.Includes.Range(func(namespace string, include ast.Include) error {
				// Set the base directory for resolving relative paths, but only if not already set
				if include.BaseDir == "" {
					include.BaseDir = dir
					tf.Includes.Set(namespace, include)
				}
				return nil
			})
		}

		err = tf.Includes.Range(func(namespace string, include ast.Include) error {
			tr := templater.Templater{Vars: tf.Vars}
			include = ast.Include{
				Namespace:      include.Namespace,
				Taskfile:       tr.Replace(include.Taskfile),
				Dir:            tr.Replace(include.Dir),
				Optional:       include.Optional,
				Internal:       include.Internal,
				Aliases:        include.Aliases,
				AdvancedImport: include.AdvancedImport,
				Vars:           include.Vars,
				BaseDir:        include.BaseDir,
			}
			if err := tr.Err(); err != nil {
				return err
			}

			uri, err := include.FullTaskfilePath()
			if err != nil {
				return err
			}

			includeReaderNode, err := NewNode(uri, insecure,
				WithParent(node),
				WithOptional(include.Optional),
			)
			if err != nil {
				if include.Optional {
					return nil
				}
				return err
			}

			if err := checkCircularIncludes(includeReaderNode); err != nil {
				return err
			}

			includedTaskfile, err := _taskfile(includeReaderNode)
			if err != nil {
				if include.Optional {
					return nil
				}
				return err
			}

			if len(includedTaskfile.Dotenv) > 0 {
				return ErrIncludedTaskfilesCantHaveDotenvs
			}

			if include.AdvancedImport {
				dir, err := include.FullDirPath()
				if err != nil {
					return err
				}

				// nolint: errcheck
				includedTaskfile.Vars.Range(func(k string, v ast.Var) error {
					o := v
					o.Dir = dir
					includedTaskfile.Vars.Set(k, o)
					return nil
				})
				// nolint: errcheck
				includedTaskfile.Env.Range(func(k string, v ast.Var) error {
					o := v
					o.Dir = dir
					includedTaskfile.Env.Set(k, o)
					return nil
				})

				for _, task := range includedTaskfile.Tasks.Values() {
					task.Dir = filepathext.SmartJoin(dir, task.Dir)
					if task.IncludeVars == nil {
						task.IncludeVars = &ast.Vars{}
					}
					task.IncludeVars.Merge(include.Vars)
					task.IncludedTaskfileVars = includedTaskfile.Vars
					task.IncludedTaskfile = &include
				}
			}

			if err = tf.Merge(includedTaskfile, &include); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return nil, err
		}

		for _, task := range tf.Tasks.Values() {
			// If the task is not defined, create a new one
			if task == nil {
				task = &ast.Task{}
			}
			// Set the location of the taskfile for each task
			if task.Location.Taskfile == "" {
				task.Location.Taskfile = tf.Location
			}
		}

		return tf, nil
	}
	return _taskfile(node)
}

func readTaskfile(
	node Node,
	download,
	offline bool,
	timeout time.Duration,
	tempDir string,
	l *logger.Logger,
) (*ast.Taskfile, error) {
	var b []byte
	var err error
	var cache *Cache

	if node.Remote() {
		cache, err = NewCache(tempDir)
		if err != nil {
			return nil, err
		}
	}

	// If the file is remote and we're in offline mode, check if we have a cached copy
	if node.Remote() && offline {
		if b, err = cache.read(node); errors.Is(err, os.ErrNotExist) {
			return nil, &errors.TaskfileCacheNotFoundError{URI: node.Location()}
		} else if err != nil {
			return nil, err
		}
		l.VerboseOutf(logger.Magenta, "task: [%s] Fetched cached copy\n", node.Location())

	} else {

		downloaded := false
		ctx, cf := context.WithTimeout(context.Background(), timeout)
		defer cf()

		// Read the file
		b, err = node.Read(ctx)
		// If we timed out then we likely have a network issue
		if node.Remote() && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			// If a download was requested, then we can't use a cached copy
			if download {
				return nil, &errors.TaskfileNetworkTimeoutError{URI: node.Location(), Timeout: timeout}
			}
			// Search for any cached copies
			if b, err = cache.read(node); errors.Is(err, os.ErrNotExist) {
				return nil, &errors.TaskfileNetworkTimeoutError{URI: node.Location(), Timeout: timeout, CheckedCache: true}
			} else if err != nil {
				return nil, err
			}
			l.VerboseOutf(logger.Magenta, "task: [%s] Network timeout. Fetched cached copy\n", node.Location())
		} else if err != nil {
			return nil, err
		} else {
			downloaded = true
		}

		// If the node was remote, we need to check the checksum
		if node.Remote() && downloaded {
			l.VerboseOutf(logger.Magenta, "task: [%s] Fetched remote copy\n", node.Location())

			// Get the checksums
			checksum := checksum(b)
			cachedChecksum := cache.readChecksum(node)

			var prompt string
			if cachedChecksum == "" {
				// If the checksum doesn't exist, prompt the user to continue
				prompt = fmt.Sprintf(taskfileUntrustedPrompt, node.Location())
			} else if checksum != cachedChecksum {
				// If there is a cached hash, but it doesn't match the expected hash, prompt the user to continue
				prompt = fmt.Sprintf(taskfileChangedPrompt, node.Location())
			}
			if prompt != "" {
				if err := l.Prompt(logger.Yellow, prompt, "n", "y", "yes"); err != nil {
					return nil, &errors.TaskfileNotTrustedError{URI: node.Location()}
				}
			}

			// If the hash has changed (or is new)
			if checksum != cachedChecksum {
				// Store the checksum
				if err := cache.writeChecksum(node, checksum); err != nil {
					return nil, err
				}
				// Cache the file
				l.VerboseOutf(logger.Magenta, "task: [%s] Caching downloaded file\n", node.Location())
				if err = cache.write(node, b); err != nil {
					return nil, err
				}
			}
		}
	}

	var t ast.Taskfile
	if err := yaml.Unmarshal(b, &t); err != nil {
		return nil, &errors.TaskfileInvalidError{URI: filepathext.TryAbsToRel(node.Location()), Err: err}
	}
	t.Location = node.Location()

	return &t, nil
}

func checkCircularIncludes(node Node) error {
	if node == nil {
		return errors.New("task: failed to check for include cycle: node was nil")
	}
	if node.Parent() == nil {
		return errors.New("task: failed to check for include cycle: node.Parent was nil")
	}
	curNode := node
	location := node.Location()
	for curNode.Parent() != nil {
		curNode = curNode.Parent()
		curLocation := curNode.Location()
		if curLocation == location {
			return fmt.Errorf("task: include cycle detected between %s <--> %s",
				curLocation,
				node.Parent().Location(),
			)
		}
	}
	return nil
}
