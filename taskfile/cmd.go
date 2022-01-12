package taskfile

// Cmd is a task command
type Cmd struct {
	Cmd         string
	Silent      bool
	Task        string
	Vars        *Vars
	IgnoreError bool
	Defer       bool
}

// Dep is a task dependency
type Dep struct {
	Task string
	Vars *Vars
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (c *Cmd) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cmd string
	if err := unmarshal(&cmd); err == nil {
		c.Cmd = cmd
		return nil
	}
	var cmdStruct struct {
		Cmd         string
		Silent      bool
		IgnoreError bool `yaml:"ignore_error"`
	}
	if err := unmarshal(&cmdStruct); err == nil && cmdStruct.Cmd != "" {
		c.Cmd = cmdStruct.Cmd
		c.Silent = cmdStruct.Silent
		c.IgnoreError = cmdStruct.IgnoreError
		return nil
	}
	var deferredCmd struct {
		Defer string
	}
	if err := unmarshal(&deferredCmd); err == nil && deferredCmd.Defer != "" {
		c.Defer = true
		c.Cmd = deferredCmd.Defer
		return nil
	}
	var deferredCall struct {
		Defer Call
	}
	if err := unmarshal(&deferredCall); err == nil && deferredCall.Defer.Task != "" {
		c.Defer = true
		c.Task = deferredCall.Defer.Task
		c.Vars = deferredCall.Defer.Vars
		return nil
	}
	var taskCall struct {
		Task string
		Vars *Vars
	}
	if err := unmarshal(&taskCall); err != nil {
		return err
	}
	c.Task = taskCall.Task
	c.Vars = taskCall.Vars
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (d *Dep) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var task string
	if err := unmarshal(&task); err == nil {
		d.Task = task
		return nil
	}
	var taskCall struct {
		Task string
		Vars *Vars
	}
	if err := unmarshal(&taskCall); err != nil {
		return err
	}
	d.Task = taskCall.Task
	d.Vars = taskCall.Vars
	return nil
}
