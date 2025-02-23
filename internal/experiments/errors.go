package experiments

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-task/task/v3/internal/slicesext"
)

type InvalidValueError struct {
	Name          string
	AllowedValues []int
	Value         int
}

func (err InvalidValueError) Error() string {
	return fmt.Sprintf(
		"task: Experiment %q has an invalid value %q (allowed values: %s)",
		err.Name,
		err.Value,
		strings.Join(slicesext.Convert(err.AllowedValues, strconv.Itoa), ", "),
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
