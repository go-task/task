package taskfile

// Tasks represents a group of tasks
type Tasks map[string]*Task

// Task represents a task
type Task struct {
	Task                 string            `json:"task"`
	Cmds                 []*Cmd            `json:"cmds"`
	Deps                 []*Dep            `json:"deps"`
	Label                string            `json:"label"`
	Desc                 string            `json:"desc"`
	Summary              string            `json:"summary"`
	Sources              []string          `json:"sources"`
	Generates            []string          `json:"generates"`
	Status               []string          `json:"status"`
	Preconditions        []*Precondition   `json:"preconditions"`
	Dir                  string            `json:"dir"`
	Vars                 *Vars             `json:"vars"`
	Env                  *Vars             `json:"env"`
	Silent               bool              `json:"silent"`
	Interactive          bool              `json:"interactive"`
	Internal             bool              `json:"internal"`
	Method               string            `json:"method"`
	Prefix               string            `json:"prefix"`
	IgnoreError          bool              `json:"ignore_error"`
	Run                  string            `json:"run"`
	IncludeVars          *Vars             `json:"include_vars"`
	IncludedTaskfileVars *Vars             `json:"included_taskfile_vars"`
	IncludedTaskfile     *IncludedTaskfile `json:"included_taskfile"`
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
