package env

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTaskEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantVal  bool
		wantOk   bool
	}{
		{"true lowercase", "true", true, true},
		{"false lowercase", "false", false, true},
		{"TRUE uppercase", "TRUE", true, true},
		{"FALSE uppercase", "FALSE", false, true},
		{"1", "1", true, true},
		{"0", "0", false, true},
		{"empty", "", false, false},
		{"invalid", "invalid", false, false},
		{"yes", "yes", false, false}, // strconv.ParseBool doesn't accept "yes"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("TASK_TEST_BOOL", tt.envValue)
			}
			val, ok := GetTaskEnvBool("TEST_BOOL")
			assert.Equal(t, tt.wantVal, val)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestGetTaskEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantVal  int
		wantOk   bool
	}{
		{"positive", "42", 42, true},
		{"zero", "0", 0, true},
		{"negative", "-5", -5, true},
		{"large", "1000000", 1000000, true},
		{"empty", "", 0, false},
		{"invalid", "abc", 0, false},
		{"float", "3.14", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("TASK_TEST_INT", tt.envValue)
			}
			val, ok := GetTaskEnvInt("TEST_INT")
			assert.Equal(t, tt.wantVal, val)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestGetTaskEnvDuration(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantVal  time.Duration
		wantOk   bool
	}{
		{"seconds", "30s", 30 * time.Second, true},
		{"minutes", "5m", 5 * time.Minute, true},
		{"hours", "2h", 2 * time.Hour, true},
		{"mixed", "1h30m", 90 * time.Minute, true},
		{"milliseconds", "500ms", 500 * time.Millisecond, true},
		{"empty", "", 0, false},
		{"invalid", "invalid", 0, false},
		{"number only", "30", 0, false}, // requires unit
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("TASK_TEST_DUR", tt.envValue)
			}
			val, ok := GetTaskEnvDuration("TEST_DUR")
			assert.Equal(t, tt.wantVal, val)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestGetTaskEnvString(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantVal  string
		wantOk   bool
	}{
		{"simple", "hello", "hello", true},
		{"with spaces", "hello world", "hello world", true},
		{"path", "/home/user/.cache", "/home/user/.cache", true},
		{"empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("TASK_TEST_STR", tt.envValue)
			}
			val, ok := GetTaskEnvString("TEST_STR")
			assert.Equal(t, tt.wantVal, val)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}

func TestGetTaskEnvStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantVal  []string
		wantOk   bool
	}{
		{"single", "github.com", []string{"github.com"}, true},
		{"multiple", "github.com,gitlab.com", []string{"github.com", "gitlab.com"}, true},
		{"with spaces", "github.com, gitlab.com , example.com", []string{"github.com", "gitlab.com", "example.com"}, true},
		{"with port", "github.com,example.com:8080", []string{"github.com", "example.com:8080"}, true},
		{"empty", "", nil, false},
		{"only commas", ",,,", nil, false},
		{"only spaces", "   ", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("TASK_TEST_SLICE", tt.envValue)
			}
			val, ok := GetTaskEnvStringSlice("TEST_SLICE")
			assert.Equal(t, tt.wantVal, val)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}
