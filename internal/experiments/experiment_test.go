package experiments_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/internal/experiments"
)

func TestNew(t *testing.T) {
	const (
		exampleExperiment    = "EXAMPLE"
		exampleExperimentEnv = "TASK_X_EXAMPLE"
	)
	tests := []struct {
		name          string
		allowedValues []int
		value         int
		wantEnabled   bool
		wantActive    bool
		wantValid     error
	}{
		{
			name:        `[] allowed, value=""`,
			wantEnabled: false,
			wantActive:  false,
		},
		{
			name:        `[] allowed, value="1"`,
			value:       1,
			wantEnabled: false,
			wantActive:  false,
			wantValid: &experiments.InactiveError{
				Name: exampleExperiment,
			},
		},
		{
			name:          `[1] allowed, value=""`,
			allowedValues: []int{1},
			wantEnabled:   false,
			wantActive:    true,
		},
		{
			name:          `[1] allowed, value="1"`,
			allowedValues: []int{1},
			value:         1,
			wantEnabled:   true,
			wantActive:    true,
		},
		{
			name:          `[1] allowed, value="2"`,
			allowedValues: []int{1},
			value:         2,
			wantEnabled:   false,
			wantActive:    true,
			wantValid: &experiments.InvalidValueError{
				Name:          exampleExperiment,
				AllowedValues: []int{1},
				Value:         2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(exampleExperimentEnv, strconv.Itoa(tt.value))
			x := experiments.New(exampleExperiment, tt.allowedValues...)
			assert.Equal(t, exampleExperiment, x.Name)
			assert.Equal(t, tt.wantEnabled, x.Enabled())
			assert.Equal(t, tt.wantActive, x.Active())
			assert.Equal(t, tt.wantValid, x.Valid())
		})
	}
}
