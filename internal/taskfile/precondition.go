package taskfile

import (
	"errors"
	"fmt"
)

var (
	// ErrCantUnmarshalPrecondition is returned for invalid precond YAML.
	ErrCantUnmarshalPrecondition = errors.New("task: can't unmarshal precondition value")
)

// Precondition represents a precondition necessary for a task to run
type Precondition struct {
	Sh          string
	Msg         string
	IgnoreError bool
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (p *Precondition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var cmd string

	if err := unmarshal(&cmd); err == nil {
		p.Sh = cmd
		p.Msg = fmt.Sprintf("`%s` failed", cmd)
		p.IgnoreError = false
		return nil
	}

	var sh struct {
		Sh          string
		Msg         string
		IgnoreError bool `yaml:"ignore_error"`
	}

	err := unmarshal(&sh)

	if err == nil {
		p.Sh = sh.Sh
		p.Msg = sh.Msg
		if p.Msg == "" {
			p.Msg = fmt.Sprintf("%s failed", sh.Sh)
		}

		p.IgnoreError = sh.IgnoreError
		return nil
	}

	return err
}
