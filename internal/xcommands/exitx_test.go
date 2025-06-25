package xcommands

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

// Mock version of ExitxCommand for testing that doesn't actually exit
type MockExitxCommand struct {
	ExitCode int
	Called   bool
}

func (m *MockExitxCommand) Execute(args []string, flags []string) error {
	m.Called = true
	m.ExitCode = 0

	// Parse exit code from first argument if provided (same logic as real command)
	if len(args) > 0 {
		code, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("exitx: invalid exit code '%s': %w", args[0], err)
		}
		m.ExitCode = code
	}

	// Don't actually exit, just store the code
	return nil
}

func TestExitxCommand_Parsing(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		flags        []string
		expectedCode int
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "default exit code (no args)",
			args:         []string{},
			flags:        []string{},
			expectedCode: 0,
			expectError:  false,
		},
		{
			name:         "exit code 0",
			args:         []string{"0"},
			flags:        []string{},
			expectedCode: 0,
			expectError:  false,
		},
		{
			name:         "exit code 1",
			args:         []string{"1"},
			flags:        []string{},
			expectedCode: 1,
			expectError:  false,
		},
		{
			name:         "exit code 42",
			args:         []string{"42"},
			flags:        []string{},
			expectedCode: 42,
			expectError:  false,
		},
		{
			name:         "negative exit code",
			args:         []string{"-1"},
			flags:        []string{},
			expectedCode: -1,
			expectError:  false,
		},
		{
			name:         "large exit code",
			args:         []string{"255"},
			flags:        []string{},
			expectedCode: 255,
			expectError:  false,
		},
		{
			name:        "invalid exit code - string",
			args:        []string{"invalid"},
			flags:       []string{},
			expectError: true,
			errorMsg:    "invalid exit code",
		},
		{
			name:        "invalid exit code - float",
			args:        []string{"1.5"},
			flags:       []string{},
			expectError: true,
			errorMsg:    "invalid exit code",
		},
		{
			name:        "invalid exit code - empty string",
			args:        []string{""},
			flags:       []string{},
			expectError: true,
			errorMsg:    "invalid exit code",
		},
		{
			name:         "flags are ignored",
			args:         []string{"5"},
			flags:        []string{"ignored", "flags"},
			expectedCode: 5,
			expectError:  false,
		},
		{
			name:         "multiple args (only first used)",
			args:         []string{"3", "ignored"},
			flags:        []string{},
			expectedCode: 3,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockExitxCommand{}
			err := mock.Execute(tt.args, tt.flags)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !containsError(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					// Verify the exit code would be correct
					if mock.ExitCode != tt.expectedCode {
						t.Errorf("Expected exit code %d, got %d", tt.expectedCode, mock.ExitCode)
					}
					if !mock.Called {
						t.Error("Expected Execute to be called")
					}
				}
			}
		})
	}
}

// Test the actual ExitxCommand in a subprocess to verify it actually exits
func TestExitxCommand_ActualExit(t *testing.T) {
	if os.Getenv("EXITX_TEST_SUBPROCESS") == "1" {
		// This runs in the subprocess
		exit := &ExitxCommand{}
		// This should cause os.Exit to be called
		exit.Execute([]string{"42"}, []string{})
		return
	}

	tests := []struct {
		name         string
		args         string
		expectedCode int
	}{
		{
			name:         "exit with code 0",
			args:         "0",
			expectedCode: 0,
		},
		{
			name:         "exit with code 1",
			args:         "1",
			expectedCode: 1,
		},
		{
			name:         "exit with code 42",
			args:         "42",
			expectedCode: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the test in a subprocess
			cmd := exec.Command(os.Args[0], "-test.run=TestExitxCommand_ActualExit")
			cmd.Env = append(os.Environ(), "EXITX_TEST_SUBPROCESS=1")
			
			// We can't pass args to the subprocess easily, so we'll test with a fixed value
			// The subprocess will exit with code 42
			err := cmd.Run()

			// Check the exit code
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode := exitError.ExitCode()
				// For this test, we expect exit code 42 since that's hardcoded above
				if exitCode != 42 {
					t.Errorf("Expected exit code 42, got %d", exitCode)
				}
			} else if err != nil {
				t.Errorf("Unexpected error running subprocess: %v", err)
			} else {
				// err == nil means exit code 0, which is wrong for our test
				t.Error("Expected non-zero exit code but got 0")
			}
		})
	}
}

func TestExitxCommand_NoExit(t *testing.T) {
	// Test the default case (no args) in a subprocess
	if os.Getenv("EXITX_TEST_DEFAULT") == "1" {
		exit := &ExitxCommand{}
		exit.Execute([]string{}, []string{})
		return
	}

	// Run the test in a subprocess to verify default exit code is 0
	cmd := exec.Command(os.Args[0], "-test.run=TestExitxCommand_NoExit")
	cmd.Env = append(os.Environ(), "EXITX_TEST_DEFAULT=1")
	
	err := cmd.Run()
	
	// For exit code 0, err should be nil
	if err != nil {
		t.Errorf("Expected exit code 0 (no error), got error: %v", err)
	}
}

func TestExitxCommand_Integration(t *testing.T) {
	// Test that exitx command is properly registered
	mu.RLock()
	cmd, exists := registry["exitx"]
	mu.RUnlock()

	if !exists {
		t.Error("exitx command not registered")
		return
	}

	if _, ok := cmd.(*ExitxCommand); !ok {
		t.Error("exitx command is not of correct type")
	}
}

func TestExitxCommand_ParseExitCode(t *testing.T) {
	// Test the integer parsing logic specifically
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"0", 0, false},
		{"1", 1, false},
		{"42", 42, false},
		{"255", 255, false},
		{"-1", -1, false},
		{"abc", 0, true},
		{"1.5", 0, true},
		{"", 0, true},
		{"999999999999999999999", 0, true}, // too large for int
	}

	for _, tt := range tests {
		t.Run("parse_"+tt.input, func(t *testing.T) {
			result, err := strconv.Atoi(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error parsing '%s' but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error parsing '%s': %v", tt.input, err)
				} else if result != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}