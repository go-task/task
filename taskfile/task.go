package taskfile

// Tasks represents a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Task                 string
	Cmds                 []*Cmd
	Deps                 []*Dep
	Label                string
	Desc                 string
	Summary              string
	Aliases              []string
	Sources              []string
	Generates            []string
	Status               []string
	Preconditions        []*Precondition
	Dir                  string
	Vars                 *Vars
	Env                  *Vars
	Silent               bool
	Interactive          bool
	Internal             bool
	Method               string
	Prefix               string
	IgnoreError          bool
	Run                  string
	IncludeVars          *Vars
	IncludedTaskfileVars *Vars
	IncludedTaskfile     *IncludedTaskfile
}

func (t *Task) Name() string {
	if t.Label != "" {
		return t.Label
	}
	return t.Task
}

func (t *Task) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cmd Cmd
	if err := unmarshal(&cmd); err == nil && cmd.Cmd != "" {
		t.Cmds = append(t.Cmds, &cmd)
		return nil
	}

	var cmds []*Cmd
	if err := unmarshal(&cmds); err == nil && len(cmds) > 0 {
		t.Cmds = cmds
		return nil
	}

	var task struct {
		Cmds          []*Cmd
		Deps          []*Dep
		Label         string
		Desc          string
		Summary       string
		Aliases       []string
		Sources       []string
		Generates     []string
		Status        []string
		Preconditions []*Precondition
		Dir           string
		Vars          *Vars
		Env           *Vars
		Silent        bool
		Interactive   bool
		Internal      bool
		Method        string
		Prefix        string
		IgnoreError   bool `yaml:"ignore_error"`
		Run           string
	}
	if err := unmarshal(&task); err != nil {
		return err
	}
	t.Cmds = task.Cmds
	t.Deps = task.Deps
	t.Label = task.Label
	t.Desc = task.Desc
	t.Aliases = task.Aliases
	t.Summary = task.Summary
	t.Sources = task.Sources
	t.Generates = task.Generates
	t.Status = task.Status
	t.Preconditions = task.Preconditions
	t.Dir = task.Dir
	t.Vars = task.Vars
	t.Env = task.Env
	t.Silent = task.Silent
	t.Interactive = task.Interactive
	t.Internal = task.Internal
	t.Method = task.Method
	t.Prefix = task.Prefix
	t.IgnoreError = task.IgnoreError
	t.Run = task.Run
	return nil
}

// DeepCopy creates a new instance of Task and copies
// data by value from the source struct.
func (t *Task) DeepCopy() *Task {
	c := &Task{
		Task:                 t.Task,
		Cmds:                 deepCopySlice(t.Cmds),
		Deps:                 deepCopySlice(t.Deps),
		Label:                t.Label,
		Desc:                 t.Desc,
		Summary:              t.Summary,
		Aliases:              deepCopySlice(t.Aliases),
		Sources:              deepCopySlice(t.Sources),
		Generates:            deepCopySlice(t.Generates),
		Status:               deepCopySlice(t.Status),
		Preconditions:        deepCopySlice(t.Preconditions),
		Dir:                  t.Dir,
		Vars:                 t.Vars.DeepCopy(),
		Env:                  t.Env.DeepCopy(),
		Silent:               t.Silent,
		Interactive:          t.Interactive,
		Internal:             t.Internal,
		Method:               t.Method,
		Prefix:               t.Prefix,
		IgnoreError:          t.IgnoreError,
		Run:                  t.Run,
		IncludeVars:          t.IncludeVars.DeepCopy(),
		IncludedTaskfileVars: t.IncludedTaskfileVars.DeepCopy(),
		IncludedTaskfile:     t.IncludedTaskfile.DeepCopy(),
	}
	return c
}
