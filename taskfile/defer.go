package taskfile

// Defer is the parameters to a defer operation.
// It can be exactly one of:
// - A string command
// - A task call
type Defer struct {
	Cmd  string
	Call *Call
}

// isValid returns true when Defer describes a valid action.
// In order for a Defer to be valid, one of Cmd or Call.Task
// must be non-empty.
func (d *Defer) isValid() bool {
	return d.Cmd != "" || (d.Call != nil && d.Call.Task != "")
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (d *Defer) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cmd string
	if err := unmarshal(&cmd); err == nil {
		d.Cmd = cmd
		return nil
	}
	var taskCall struct {
		Task string
		Vars *Vars
	}
	if err := unmarshal(&taskCall); err != nil {
		return err
	}
	d.Call = &Call{
		Task: taskCall.Task,
		Vars: taskCall.Vars,
	}
	return nil
}
