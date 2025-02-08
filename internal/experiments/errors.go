package experiments

import (
	"fmt"
	"strings"
)

type ExperimentHasInvalidValueError struct {
	x Experiment
}

func (err ExperimentHasInvalidValueError) Error() string {
	return fmt.Sprintf(
		"task: Experiment %q has an invalid value %q (allowed values: %s)",
		err.x.Name,
		err.x.Value,
		strings.Join(err.x.AllowedValues, ", "),
	)
}

type ExperimentInactiveError struct {
	x Experiment
}

func (err ExperimentInactiveError) Error() string {
	return fmt.Sprintf(
		"task: Experiment %q is inactive and cannot be enabled",
		err.x.Name,
	)
}
