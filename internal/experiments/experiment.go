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

// New creates a new experiment with the given name and sets the values that can
// enable it.
func New(xName string, allowedValues ...string) Experiment {
	value := getEnv(xName)
	x := Experiment{
		Name:          xName,
		AllowedValues: allowedValues,
		Value:         value,
	}
	xList = append(xList, x)
	return x
}

func (x *Experiment) Enabled() bool {
	return slices.Contains(x.AllowedValues, x.Value)
}

func (x *Experiment) Active() bool {
	return len(x.AllowedValues) > 0
}

func (x Experiment) Valid() error {
	if !x.Active() && x.Value != "" {
		return &InactiveError{
			Name: x.Name,
		}
	}
	if !x.Enabled() && x.Value != "" {
		return &InvalidValueError{
			Name:          x.Name,
			AllowedValues: x.AllowedValues,
			Value:         x.Value,
		}
	}
	return nil
}

func (x Experiment) String() string {
	if x.Enabled() {
		return fmt.Sprintf("on (%s)", x.Value)
	}
	return "off"
}
