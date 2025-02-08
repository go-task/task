package experiments

import (
	"fmt"
	"slices"
)

type Experiment struct {
	Name          string   // The name of the experiment.
	AllowedValues []string // The values that can enable this experiment.
	Value         string   // The version of the experiment that is enabled.
}

func (x *Experiment) Enabled() bool {
	return slices.Contains(x.AllowedValues, x.Value)
}

func (x *Experiment) Active() bool {
	return len(x.AllowedValues) > 0
}

func (x Experiment) Valid() error {
	if !x.Active() && x.Value != "" {
		return ExperimentInactiveError{x}
	}
	if !x.Enabled() && x.Value != "" {
		return ExperimentHasInvalidValueError{x}
	}
	return nil
}

func (x Experiment) String() string {
	if x.Enabled() {
		return fmt.Sprintf("on (%s)", x.Value)
	}
	return "off"
}
