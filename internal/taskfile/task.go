package taskfile

// Tasks representas a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Task        string
	Cmds        []*Cmd
	Deps        []*Dep
	Desc        string
	Sources     []string
	Generates   []string
	Status      []string
	Dir         string
	Vars        Vars
	Env         Vars
	Silent      bool
	Method      string
	Prefix      string
	IgnoreError bool `yaml:"ignore_error"`
}
