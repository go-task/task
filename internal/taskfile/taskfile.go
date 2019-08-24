package taskfile

// Taskfile represents a Taskfile.yml
type Taskfile struct {
	Version    string
	Expansions int
	Output     string
	Includes   map[string]string
	Vars       Vars
	Env        Vars
	Tasks      Tasks
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (tf *Taskfile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&tf.Tasks); err == nil {
		tf.Version = "1"
		return nil
	}

	var taskfile struct {
		Version    string
		Expansions int
		Output     string
		Includes   map[string]string
		Vars       Vars
		Env        Vars
		Tasks      Tasks
	}
	if err := unmarshal(&taskfile); err != nil {
		return err
	}
	tf.Version = taskfile.Version
	tf.Expansions = taskfile.Expansions
	tf.Output = taskfile.Output
	tf.Includes = taskfile.Includes
	tf.Vars = taskfile.Vars
	tf.Env = taskfile.Env
	tf.Tasks = taskfile.Tasks
	if tf.Expansions <= 0 {
		tf.Expansions = 2
	}
	if tf.Vars == nil {
		tf.Vars = make(Vars)
	}
	return nil
}

func (t *Task) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var task struct {
		Task          string
		Cmds          []*Cmd
		Deps          []*Dep
		Desc          string
		Summary       string
		Sources       []string
		Generates     []string
		Status        []string
		Preconditions []*Precondition
		Dir           string
		Vars          Vars
		Env           Vars
		Silent        bool
		Method        string
		Prefix        string
		IgnoreError   bool `yaml:"ignore_error"`
	}

	err := unmarshal(&task)
	if err != nil {
		var short []string
		err := unmarshal(&short)
		if err != nil {
			return err
		}

		for _, cmd := range short {
			t.Cmds = append(t.Cmds, &Cmd{
				Cmd: cmd,
			})

		}

		return nil
	}

	t.Task = task.Task
	t.Cmds = task.Cmds
	t.Deps = task.Deps
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
	t.Method = task.Method
	t.Prefix = task.Prefix
	t.IgnoreError = task.IgnoreError

	return nil
}
