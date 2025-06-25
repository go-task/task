package xcommands

import (
	"testing"
	"time"
)

func TestSleepxCommand_Execute(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         []string
		expectError   bool
		expectedError string
		minDuration   time.Duration
		maxDuration   time.Duration
	}{
		{
			name:          "sleep for 1 millisecond",
			args:          []string{"1ms"},
			flags:         []string{},
			expectError:   false,
			minDuration:   1 * time.Millisecond,
			maxDuration:   100 * time.Millisecond, // Allow some overhead
		},
		{
			name:          "sleep for fractional seconds",
			args:          []string{"0.001"}, // 1 millisecond as float
			flags:         []string{},
			expectError:   false,
			minDuration:   1 * time.Millisecond,
			maxDuration:   100 * time.Millisecond,
		},
		{
			name:          "sleep for integer seconds",
			args:          []string{"0"}, // 0 seconds
			flags:         []string{},
			expectError:   false,
			minDuration:   0,
			maxDuration:   50 * time.Millisecond,
		},
		{
			name:          "sleep with Go duration format",
			args:          []string{"10ms"},
			flags:         []string{},
			expectError:   false,
			minDuration:   10 * time.Millisecond,
			maxDuration:   100 * time.Millisecond,
		},
		{
			name:          "missing operand",
			args:          []string{},
			flags:         []string{},
			expectError:   true,
			expectedError: "missing operand",
		},
		{
			name:          "too many arguments",
			args:          []string{"1", "2"},
			flags:         []string{},
			expectError:   true,
			expectedError: "too many arguments",
		},
		{
			name:          "invalid duration format",
			args:          []string{"invalid"},
			flags:         []string{},
			expectError:   true,
			expectedError: "invalid time interval",
		},
		{
			name:          "negative duration",
			args:          []string{"-1"},
			flags:         []string{},
			expectError:   true,
			expectedError: "negative duration",
		},
		{
			name:          "empty string duration",
			args:          []string{""},
			flags:         []string{},
			expectError:   true,
			expectedError: "invalid time interval",
		},
		{
			name:          "flags are ignored",
			args:          []string{"1ms"},
			flags:         []string{"ignored", "flags"},
			expectError:   false,
			minDuration:   1 * time.Millisecond,
			maxDuration:   100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sleep := &SleepxCommand{}
			
			start := time.Now()
			err := sleep.Execute(tt.args, tt.flags)
			elapsed := time.Since(start)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.expectedError != "" && !containsError(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					// Verify sleep duration
					if elapsed < tt.minDuration {
						t.Errorf("Sleep was too short: expected at least %v, got %v", tt.minDuration, elapsed)
					}
					if elapsed > tt.maxDuration {
						t.Errorf("Sleep was too long: expected at most %v, got %v", tt.maxDuration, elapsed)
					}
				}
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "Go duration - milliseconds",
			input:    "100ms",
			expected: 100 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:     "Go duration - seconds",
			input:    "2s",
			expected: 2 * time.Second,
			wantErr:  false,
		},
		{
			name:     "Go duration - minutes",
			input:    "1m",
			expected: 1 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "Go duration - hours",
			input:    "1h",
			expected: 1 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "Go duration - complex",
			input:    "1h30m",
			expected: 1*time.Hour + 30*time.Minute,
			wantErr:  false,
		},
		{
			name:     "integer seconds",
			input:    "5",
			expected: 5 * time.Second,
			wantErr:  false,
		},
		{
			name:     "zero seconds",
			input:    "0",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "fractional seconds",
			input:    "0.5",
			expected: 500 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:     "small fractional seconds",
			input:    "0.001",
			expected: 1 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "mixed invalid format",
			input:   "1x",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDuration(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestSleepxCommand_Integration(t *testing.T) {
	// Test that sleepx command is properly registered
	mu.RLock()
	cmd, exists := registry["sleepx"]
	mu.RUnlock()

	if !exists {
		t.Error("sleepx command not registered")
		return
	}

	if _, ok := cmd.(*SleepxCommand); !ok {
		t.Error("sleepx command is not of correct type")
	}
}

// Helper function to check if error message contains expected text
func containsError(errorMsg, expected string) bool {
	return len(errorMsg) > 0 && len(expected) > 0 && 
		   (errorMsg == expected || 
		    (len(errorMsg) > len(expected) && 
		     (errorMsg[:len(expected)] == expected || 
		      errorMsg[len(errorMsg)-len(expected):] == expected ||
		      containsSubstring(errorMsg, expected))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}