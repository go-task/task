package taskfile

// Tasks represents a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Task         string
	Cmds         []*Cmd
	Deps         []*Dep
	Desc         string
	Summary      string
	Sources      []string
	Generates    []string
	Status       []string
	Preconditions []*Precondition
	Dir          string
	Vars         Vars
	Env          Vars
	Silent       bool
	Method       string
	Prefix       string
	IgnoreError  bool
}
