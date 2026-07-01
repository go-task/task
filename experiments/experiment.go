package experiments

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/go-task/task/v3/taskrc/ast"
)

type Experiment struct {
	Name           string // The name of the experiment.
	AllowedValues  []int  // The values that can enable this experiment.
	Value          int    // The version of the experiment that is enabled.
	InactiveReason string // If not active, the reason why it is inactive.
}

func getValue(xName string, config *ast.TaskRC) int {
	var value int
	if config != nil {
		value = config.Experiments[xName]
	}
	if value == 0 {
		value, _ = strconv.Atoi(getEnv(xName))
	}
	return value
}

// New creates a new experiment with the given name and sets the values that can
// enable it.
func New(xName string, config *ast.TaskRC, allowedValues ...int) Experiment {
	x := Experiment{
		Name:          xName,
		AllowedValues: allowedValues,
		Value:         getValue(xName, config),
	}
	xList = append(xList, x)
	return x
}

// NewStable creates a new experiment that is stable and no longer needs to be
// enabled. It will always be inactive and cannot be enabled.
func NewStable(xName string, config *ast.TaskRC) Experiment {
	x := Experiment{
		Name:           xName,
		Value:          getValue(xName, config),
		InactiveReason: "is stable and no longer needs to be enabled",
	}
	xList = append(xList, x)
	return x
}

// NewAbandoned creates a new experiment that has been abandoned and is no
// longer supported. It will always be inactive and cannot be enabled.
func NewAbandoned(xName string, config *ast.TaskRC) Experiment {
	x := Experiment{
		Name:           xName,
		Value:          getValue(xName, config),
		InactiveReason: "has been abandoned and is no longer supported",
	}
	xList = append(xList, x)
	return x
}

func (x Experiment) Enabled() bool {
	return slices.Contains(x.AllowedValues, x.Value)
}

func (x Experiment) Active() bool {
	return len(x.AllowedValues) > 0
}

func (x Experiment) Valid() error {
	if !x.Active() && x.Value != 0 {
		return &InactiveError{
			Name:   x.Name,
			Reason: x.InactiveReason,
		}
	}
	if !x.Enabled() && x.Value != 0 {
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
		return fmt.Sprintf("on (%d)", x.Value)
	}
	return "off"
}
