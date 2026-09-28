package flags_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3"
	"github.com/go-task/task/v3/internal/flags"
)

// WithFlags applies WithTaskSorter after NewExecutor set its default, so an
// unset --sort must still resolve to a sorter instead of clearing it.
func TestWithFlags_DefaultSorterIsNotCleared(t *testing.T) { //nolint:paralleltest // mutates package state
	original := flags.TaskSort
	t.Cleanup(func() { flags.TaskSort = original })

	for _, sort := range []string{"", "default"} {
		flags.TaskSort = sort
		e := task.NewExecutor(flags.WithFlags())
		require.NotNilf(t, e.TaskSorter, "--sort %q left the executor without a sorter", sort)
	}
}
