package experiments

import (
	"fmt"
	"strings"
)

type InvalidValueError struct {
	Name          string
	AllowedValues []string
	Value         string
}

func (err InvalidValueError) Error() string {
	return fmt.Sprintf(
		"task: Experiment %q has an invalid value %q (allowed values: %s)",
		err.Name,
		err.Value,
		strings.Join(err.AllowedValues, ", "),
	)
}

type InactiveError struct {
	Name string
}

func (err InactiveError) Error() string {
	return fmt.Sprintf(
		"task: Experiment %q is inactive and cannot be enabled",
		err.Name,
	)
}
