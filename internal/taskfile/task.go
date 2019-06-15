package taskfile

import "os"
import "sync"

// Tasks represents a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Task        string
	Cmds        []*Cmd
	Deps        []*Dep
	Desc        string
	Summary     string
	Sources     []string
	Generates   []string
	Status      []string
	Dir         string
	mkdirMutex  sync.Mutex
	Vars        Vars
	Env         Vars
	Silent      bool
	Method      string
	Prefix      string
	IgnoreError bool `yaml:"ignore_error"`
}

// Mkdir creates the directory Task.Dir.
// Safe to be called concurrently.
func (t *Task) Mkdir() error {
	if t.Dir == "" {
		// No "dir:" attribute, so we do nothing.
		return nil
	}

	t.mkdirMutex.Lock()
	defer t.mkdirMutex.Unlock()

	if _, err := os.Stat(t.Dir); os.IsNotExist(err) {
		if err := os.MkdirAll(t.Dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
