package experiments_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/experiments"
	"github.com/go-task/task/v3/taskrc/ast"
)

func TestNew(t *testing.T) {
	const (
		exampleExperiment    = "EXAMPLE"
		exampleExperimentEnv = "TASK_X_EXAMPLE"
	)
	tests := []struct {
		name          string
		config        *ast.TaskRC
		allowedValues []int
		env           int
		wantEnabled   bool
		wantActive    bool
		wantValid     error
		wantValue     int
	}{
		{
			name:        `[] allowed, env=""`,
			wantEnabled: false,
			wantActive:  false,
		},
		{
			name:        `[] allowed, env="1"`,
			env:         1,
			wantEnabled: false,
			wantActive:  false,
			wantValid: &experiments.InactiveError{
				Name: exampleExperiment,
			},
			wantValue: 1,
		},
		{
			name:          `[1] allowed, env=""`,
			allowedValues: []int{1},
			wantEnabled:   false,
			wantActive:    true,
		},
		{
			name:          `[1] allowed, env="1"`,
			allowedValues: []int{1},
			env:           1,
			wantEnabled:   true,
			wantActive:    true,
			wantValue:     1,
		},
		{
			name:          `[1] allowed, env="2"`,
			allowedValues: []int{1},
			env:           2,
			wantEnabled:   false,
			wantActive:    true,
			wantValid: &experiments.InvalidValueError{
				Name:          exampleExperiment,
				AllowedValues: []int{1},
				Value:         2,
			},
			wantValue: 2,
		},
		{
			name:          `[1, 2] allowed, env="1"`,
			allowedValues: []int{1, 2},
			env:           1,
			wantEnabled:   true,
			wantActive:    true,
			wantValue:     1,
		},
		{
			name:          `[1, 2] allowed, env="1"`,
			allowedValues: []int{1, 2},
			env:           2,
			wantEnabled:   true,
			wantActive:    true,
			wantValue:     2,
		},
		{
			name: `[1] allowed, config="1"`,
			config: &ast.TaskRC{
				Experiments: map[string]int{
					exampleExperiment: 1,
				},
			},
			allowedValues: []int{1},
			wantEnabled:   true,
			wantActive:    true,
			wantValue:     1,
		},
		{
			name: `[1] allowed, config="2"`,
			config: &ast.TaskRC{
				Experiments: map[string]int{
					exampleExperiment: 2,
				},
			},
			allowedValues: []int{1},
			wantEnabled:   false,
			wantActive:    true,
			wantValid: &experiments.InvalidValueError{
				Name:          exampleExperiment,
				AllowedValues: []int{1},
				Value:         2,
			},
			wantValue: 2,
		},
		{
			name: `[1, 2] allowed, env="1", config="2"`,
			config: &ast.TaskRC{
				Experiments: map[string]int{
					exampleExperiment: 2,
				},
			},
			allowedValues: []int{1, 2},
			env:           1,
			wantEnabled:   true,
			wantActive:    true,
			wantValue:     2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(exampleExperimentEnv, strconv.Itoa(tt.env))
			x := experiments.New(exampleExperiment, tt.config, tt.allowedValues...)
			assert.Equal(t, exampleExperiment, x.Name)
			assert.Equal(t, tt.wantEnabled, x.Enabled())
			assert.Equal(t, tt.wantActive, x.Active())
			assert.Equal(t, tt.wantValid, x.Valid())
			assert.Equal(t, tt.wantValue, x.Value)
		})
	}
}
