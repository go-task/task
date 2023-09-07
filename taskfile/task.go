package taskfile

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/go-task/task/v3/internal/deepcopy"
)

// Task represents a task
type Task struct {
	Task                 string
	Cmds                 []*Cmd
	Deps                 []*Dep
	Label                string
	Desc                 string
	Prompt               string
	Summary              string
	Requires             *Requires
	Aliases              []string
	Sources              []*Glob
	Generates            []*Glob
	Status               []string
	Preconditions        []*Precondition
	Dir                  string
	Set                  []string
	Shopt                []string
	Vars                 *Vars
	Env                  *Vars
	Dotenv               []string
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
	Platforms            []*Platform
	Location             *Location
	Watch                bool
}

func (t *Task) Name() string {
	if t.Label != "" {
		return t.Label
	}
	return t.Task
}

func (t *Task) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	// Shortcut syntax for a task with a single command
	case yaml.ScalarNode:
		var cmd Cmd
		if err := node.Decode(&cmd); err != nil {
			return err
		}
		t.Cmds = append(t.Cmds, &cmd)
		return nil

	// Shortcut syntax for a simple task with a list of commands
	case yaml.SequenceNode:
		var cmds []*Cmd
		if err := node.Decode(&cmds); err != nil {
			return err
		}
		t.Cmds = cmds
		return nil

	// Full task object
	case yaml.MappingNode:
		var task struct {
			Cmds          []*Cmd
			Cmd           *Cmd
			Deps          []*Dep
			Label         string
			Desc          string
			Prompt        string
			Summary       string
			Aliases       []string
			Sources       []*Glob
			Generates     []*Glob
			Status        []string
			Preconditions []*Precondition
			Dir           string
			Set           []string
			Shopt         []string
			Vars          *Vars
			Env           *Vars
			Dotenv        []string
			Silent        bool
			Interactive   bool
			Internal      bool
			Method        string
			Prefix        string
			IgnoreError   bool `yaml:"ignore_error"`
			Run           string
			Platforms     []*Platform
			Requires      *Requires
			Watch         bool
		}
		if err := node.Decode(&task); err != nil {
			return err
		}
		if task.Cmd != nil {
			if task.Cmds != nil {
				return fmt.Errorf("yaml: line %d: task cannot have both cmd and cmds", node.Line)
			}
			t.Cmds = []*Cmd{task.Cmd}
		} else {
			t.Cmds = task.Cmds
		}
		t.Deps = task.Deps
		t.Label = task.Label
		t.Desc = task.Desc
		t.Prompt = task.Prompt
		t.Summary = task.Summary
		t.Aliases = task.Aliases
		t.Sources = task.Sources
		t.Generates = task.Generates
		t.Status = task.Status
		t.Preconditions = task.Preconditions
		t.Dir = task.Dir
		t.Set = task.Set
		t.Shopt = task.Shopt
		t.Vars = task.Vars
		t.Env = task.Env
		t.Dotenv = task.Dotenv
		t.Silent = task.Silent
		t.Interactive = task.Interactive
		t.Internal = task.Internal
		t.Method = task.Method
		t.Prefix = task.Prefix
		t.IgnoreError = task.IgnoreError
		t.Run = task.Run
		t.Platforms = task.Platforms
		t.Requires = task.Requires
		t.Watch = task.Watch
		return nil
	}

	return fmt.Errorf("yaml: line %d: cannot unmarshal %s into task", node.Line, node.ShortTag())
}

// DeepCopy creates a new instance of Task and copies
// data by value from the source struct.
func (t *Task) DeepCopy() *Task {
	if t == nil {
		return nil
	}
	c := &Task{
		Task:                 t.Task,
		Cmds:                 deepcopy.Slice(t.Cmds),
		Deps:                 deepcopy.Slice(t.Deps),
		Label:                t.Label,
		Desc:                 t.Desc,
		Prompt:               t.Prompt,
		Summary:              t.Summary,
		Aliases:              deepcopy.Slice(t.Aliases),
		Sources:              deepcopy.Slice(t.Sources),
		Generates:            deepcopy.Slice(t.Generates),
		Status:               deepcopy.Slice(t.Status),
		Preconditions:        deepcopy.Slice(t.Preconditions),
		Dir:                  t.Dir,
		Set:                  deepcopy.Slice(t.Set),
		Shopt:                deepcopy.Slice(t.Shopt),
		Vars:                 t.Vars.DeepCopy(),
		Env:                  t.Env.DeepCopy(),
		Dotenv:               deepcopy.Slice(t.Dotenv),
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
		Platforms:            deepcopy.Slice(t.Platforms),
		Location:             t.Location.DeepCopy(),
		Requires:             t.Requires.DeepCopy(),
	}
	return c
}
