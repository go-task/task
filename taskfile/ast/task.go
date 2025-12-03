package ast

import (
	"fmt"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/go-task/task/v3/errors"
	"github.com/go-task/task/v3/internal/deepcopy"
)

// Task represents a task
type Task struct {
	Task          string `hash:"ignore"`
	Cmds          []*Cmd
	Deps          []*Dep
	Label         string
	Desc          string
	Prompt        Prompt
	Summary       string
	Requires      *Requires
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
	Prefix        string `hash:"ignore"`
	IgnoreError   bool
	Run           string
	Platforms     []*Platform
	Watch         bool
	Location      *Location
	Failfast      bool
	// Populated during merging
	Namespace            string `hash:"ignore"`
	IncludeVars          *Vars
	IncludedTaskfileVars *Vars

	FullName string `hash:"ignore"`
}

func (t *Task) Name() string {
	if t.Label != "" {
		return t.Label
	}
	if t.FullName != "" {
		return t.FullName
	}
	return t.Task
}

func (t *Task) LocalName() string {
	name := t.FullName
	name = strings.TrimPrefix(name, t.Namespace)
	name = strings.TrimPrefix(name, ":")
	return name
}

// WildcardMatch will check if the given string matches the name of the Task and returns any wildcard values.
func (t *Task) WildcardMatch(name string) (bool, []string) {
	// Convert the name into a regex string
	regexStr := fmt.Sprintf("^%s$", strings.ReplaceAll(t.Task, "*", "(.*)"))
	regex := regexp.MustCompile(regexStr)
	wildcards := regex.FindStringSubmatch(name)
	wildcardCount := strings.Count(t.Task, "*")

	// If there are no wildcards, return false
	if len(wildcards) == 0 {
		return false, nil
	}

	// Remove the first match, which is the full string
	wildcards = wildcards[1:]

	// If there are more/less wildcards than matches, return false
	if len(wildcards) != wildcardCount {
		return false, wildcards
	}

	return true, wildcards
}

func (t *Task) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {

	// Shortcut syntax for a task with a single command
	case yaml.ScalarNode:
		var cmd Cmd
		if err := node.Decode(&cmd); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		t.Cmds = append(t.Cmds, &cmd)
		return nil

	// Shortcut syntax for a simple task with a list of commands
	case yaml.SequenceNode:
		var cmds []*Cmd
		if err := node.Decode(&cmds); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
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
			Prompt        Prompt
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
			Failfast      bool
		}
		if err := node.Decode(&task); err != nil {
			return errors.NewTaskfileDecodeError(err, node)
		}
		if task.Cmd != nil {
			if task.Cmds != nil {
				return errors.NewTaskfileDecodeError(nil, node).WithMessage("task cannot have both cmd and cmds")
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
		t.Failfast = task.Failfast
		return nil
	}

	return errors.NewTaskfileDecodeError(nil, node).WithTypeMessage("task")
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
		Platforms:            deepcopy.Slice(t.Platforms),
		Location:             t.Location.DeepCopy(),
		Requires:             t.Requires.DeepCopy(),
		Namespace:            t.Namespace,
		FullName:             t.FullName,
		Failfast:             t.Failfast,
	}
	return c
}
