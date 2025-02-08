package experiments_test

import (
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
		allowedValues []string
		value         string
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
			value:       "1",
			wantEnabled: false,
			wantActive:  false,
			wantValid: &experiments.InactiveError{
				Name: exampleExperiment,
			},
		},
		{
			name:          `[1] allowed, value=""`,
			allowedValues: []string{"1"},
			wantEnabled:   false,
			wantActive:    true,
		},
		{
			name:          `[1] allowed, value="1"`,
			allowedValues: []string{"1"},
			value:         "1",
			wantEnabled:   true,
			wantActive:    true,
		},
		{
			name:          `[1] allowed, value="2"`,
			allowedValues: []string{"1"},
			value:         "2",
			wantEnabled:   false,
			wantActive:    true,
			wantValid: &experiments.InvalidValueError{
				Name:          exampleExperiment,
				AllowedValues: []string{"1"},
				Value:         "2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(exampleExperimentEnv, tt.value)
			x := experiments.New(exampleExperiment, tt.allowedValues...)
			assert.Equal(t, exampleExperiment, x.Name)
			assert.Equal(t, tt.wantEnabled, x.Enabled())
			assert.Equal(t, tt.wantActive, x.Active())
			assert.Equal(t, tt.wantValid, x.Valid())
		})
	}
}
