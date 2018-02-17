package taskfile

// Taskfile represents a Taskfile.yml
type Taskfile struct {
	// TODO: version is still not used
	Version int
	Tasks   Tasks
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (tf *Taskfile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&tf.Tasks); err == nil {
		return nil
	}

	var taskfile struct {
		Version int
		Tasks   Tasks
	}
	if err := unmarshal(&taskfile); err != nil {
		return err
	}
	tf.Version = taskfile.Version
	tf.Tasks = taskfile.Tasks
	return nil
}
