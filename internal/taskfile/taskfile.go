package taskfile

// Taskfile represents a Taskfile.yml
type Taskfile struct {
	Version string
	Vars    Vars
	Tasks   Tasks
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (tf *Taskfile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&tf.Tasks); err == nil {
		tf.Version = "1"
		return nil
	}

	var taskfile struct {
		Version string
		Vars    Vars
		Tasks   Tasks
	}
	if err := unmarshal(&taskfile); err != nil {
		return err
	}
	tf.Version = taskfile.Version
	tf.Vars = taskfile.Vars
	tf.Tasks = taskfile.Tasks
	return nil
}
