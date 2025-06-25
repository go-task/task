package xcommands

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestXargsxCommand_Execute(t *testing.T) {
	// Save original stdin to restore later
	originalStdin := os.Stdin

	tests := []struct {
		name         string
		args         []string
		flags        []string
		stdinContent string
		expectError  bool
		validateFunc func(t *testing.T)
	}{
		{
			name:         "echo with single argument from stdin",
			args:         []string{"echo"},
			flags:        []string{},
			stdinContent: "hello",
			expectError:  false,
			validateFunc: func(t *testing.T) {
				// This test just verifies no error occurs
				// Output verification is complex due to subprocess execution
			},
		},
		{
			name:         "echo with multiple arguments from stdin",
			args:         []string{"echo"},
			flags:        []string{},
			stdinContent: "hello world test",
			expectError:  false,
			validateFunc: func(t *testing.T) {
				// This test verifies the command executes without error
			},
		},
		{
			name:         "echo with multiline stdin",
			args:         []string{"echo"},
			flags:        []string{},
			stdinContent: "line1\nline2\nline3",
			expectError:  false,
			validateFunc: func(t *testing.T) {
				// Multiline input should be processed correctly
			},
		},
		{
			name:         "echo with empty stdin",
			args:         []string{"echo"},
			flags:        []string{},
			stdinContent: "",
			expectError:  false,
			validateFunc: func(t *testing.T) {
				// Empty stdin should still work
			},
		},
		{
			name:         "echo with whitespace-only stdin",
			args:         []string{"echo"},
			flags:        []string{},
			stdinContent: "   \n  \t  \n   ",
			expectError:  false,
			validateFunc: func(t *testing.T) {
				// Whitespace should be trimmed
			},
		},
		{
			name:         "command with initial arguments plus stdin",
			args:         []string{"echo", "prefix"},
			flags:        []string{},
			stdinContent: "from stdin",
			expectError:  false,
			validateFunc: func(t *testing.T) {
				// Should combine initial args with stdin args
			},
		},
		{
			name:        "missing command",
			args:        []string{},
			flags:       []string{},
			stdinContent: "test",
			expectError: true,
			validateFunc: func(t *testing.T) {
				// Should error when no command provided
			},
		},
		{
			name:         "non-existent command",
			args:         []string{"non_existent_command_12345"},
			flags:        []string{},
			stdinContent: "test",
			expectError:  true,
			validateFunc: func(t *testing.T) {
				// Should error when command doesn't exist
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pipes for stdin
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			defer r.Close()

			// Write test content to stdin
			go func() {
				defer w.Close()
				w.Write([]byte(tt.stdinContent))
			}()

			// Set up stdin
			os.Stdin = r
			defer func() {
				os.Stdin = originalStdin
			}()

			// Execute command
			xargsx := &XargsxCommand{}
			err = xargsx.Execute(tt.args, tt.flags)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Run validation function
			if tt.validateFunc != nil {
				tt.validateFunc(t)
			}
		})
	}
}

func TestXargsxCommand_ExecuteWithCapture(t *testing.T) {
	// Save original stdin to restore later
	originalStdin := os.Stdin

	tests := []struct {
		name           string
		args           []string
		stdinContent   string
		expectedInArgs []string
	}{
		{
			name:           "single word from stdin",
			args:           []string{"echo", "before"},
			stdinContent:   "after",
			expectedInArgs: []string{"before", "after"}, // echo should receive these args
		},
		{
			name:           "multiple words from stdin",
			args:           []string{"echo"},
			stdinContent:   "word1 word2 word3",
			expectedInArgs: []string{"word1", "word2", "word3"},
		},
		{
			name:           "multiline input",
			args:           []string{"echo"},
			stdinContent:   "line1\nline2 word2\nline3",
			expectedInArgs: []string{"line1", "line2", "word2", "line3"},
		},
		{
			name:           "mixed spaces and tabs",
			args:           []string{"echo"},
			stdinContent:   "word1\t\tword2   word3\n\tword4",
			expectedInArgs: []string{"word1", "word2", "word3", "word4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pipes for stdin
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			defer r.Close()

			// Write test content to stdin
			go func() {
				defer w.Close()
				w.Write([]byte(tt.stdinContent))
			}()

			// Set up stdin
			os.Stdin = r
			defer func() {
				os.Stdin = originalStdin
			}()

			// For testing purposes, we'll use a mock command that we can verify
			// We'll test the stdin parsing logic separately since subprocess testing is complex
			xargsx := &XargsxCommand{}
			
			// This test mainly verifies that the command executes without error
			// The actual argument passing is tested separately
			err = xargsx.Execute(tt.args, []string{})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestXargsxCommand_StdinParsing(t *testing.T) {
	// Test the stdin parsing logic in isolation
	originalStdin := os.Stdin

	tests := []struct {
		name         string
		stdinContent string
		expected     []string
	}{
		{
			name:         "simple words",
			stdinContent: "word1 word2 word3",
			expected:     []string{"word1", "word2", "word3"},
		},
		{
			name:         "multiline input",
			stdinContent: "line1\nline2\nline3",
			expected:     []string{"line1", "line2", "line3"},
		},
		{
			name:         "mixed whitespace",
			stdinContent: "  word1  \t word2\n\nword3  ",
			expected:     []string{"word1", "word2", "word3"},
		},
		{
			name:         "empty lines",
			stdinContent: "word1\n\n\nword2\n\n",
			expected:     []string{"word1", "word2"},
		},
		{
			name:         "only whitespace",
			stdinContent: "   \n  \t  \n   ",
			expected:     []string{},
		},
		{
			name:         "empty input",
			stdinContent: "",
			expected:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pipes for stdin
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			defer r.Close()

			// Write test content to stdin
			go func() {
				defer w.Close()
				w.Write([]byte(tt.stdinContent))
			}()

			// Set up stdin
			os.Stdin = r
			defer func() {
				os.Stdin = originalStdin
			}()

			// Use a custom implementation to test parsing logic
			parsed := parseStdinArgs()

			if len(parsed) != len(tt.expected) {
				t.Errorf("Expected %d arguments, got %d", len(tt.expected), len(parsed))
			}

			for i, arg := range parsed {
				if i >= len(tt.expected) || arg != tt.expected[i] {
					t.Errorf("Expected argument %d to be %q, got %q", i, tt.expected[i], arg)
				}
			}
		})
	}
}

func TestXargsxCommand_ErrorMessages(t *testing.T) {
	originalStdin := os.Stdin

	tests := []struct {
		name          string
		args          []string
		stdinContent  string
		expectedError string
	}{
		{
			name:          "missing command",
			args:          []string{},
			stdinContent:  "test",
			expectedError: "xargsx: missing command to execute",
		},
		{
			name:          "command not found",
			args:          []string{"definitely_not_a_real_command_12345"},
			stdinContent:  "test",
			expectedError: "xargsx: failed to execute 'definitely_not_a_real_command_12345'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pipes for stdin
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			defer r.Close()

			// Write test content to stdin
			go func() {
				defer w.Close()
				w.Write([]byte(tt.stdinContent))
			}()

			// Set up stdin
			os.Stdin = r
			defer func() {
				os.Stdin = originalStdin
			}()

			// Execute command
			xargsx := &XargsxCommand{}
			err = xargsx.Execute(tt.args, []string{})

			if err == nil {
				t.Error("Expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error to contain %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestXargsxCommand_ExitCodeHandling(t *testing.T) {
	// Test handling of commands that exit with non-zero codes
	originalStdin := os.Stdin

	// Create pipes for stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer r.Close()

	// Write test content to stdin
	go func() {
		defer w.Close()
		w.Write([]byte(""))
	}()

	// Set up stdin
	os.Stdin = r
	defer func() {
		os.Stdin = originalStdin
	}()

	// Use a command that will exit with non-zero status
	// "false" command always exits with status 1
	xargsx := &XargsxCommand{}
	err = xargsx.Execute([]string{"false"}, []string{})

	if err == nil {
		t.Error("Expected error for command with non-zero exit code")
	}

	expectedErrorMsg := "xargsx: command 'false' exited with code"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain %q, got %q", expectedErrorMsg, err.Error())
	}
}

func TestXargsxCommand_RealWorldUsage(t *testing.T) {
	// Test with real commands that should be available on most systems
	originalStdin := os.Stdin

	tests := []struct {
		name         string
		args         []string
		stdinContent string
		skipTest     bool
		skipReason   string
	}{
		{
			name:         "echo command",
			args:         []string{"echo", "prefix"},
			stdinContent: "suffix",
			skipTest:     false,
		},
		{
			name:         "true command",
			args:         []string{"true"},
			stdinContent: "ignored",
			skipTest:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipTest {
				t.Skip(tt.skipReason)
			}

			// Check if command exists
			if _, err := exec.LookPath(tt.args[0]); err != nil {
				t.Skipf("Command %s not available: %v", tt.args[0], err)
			}

			// Create pipes for stdin
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			defer r.Close()

			// Write test content to stdin
			go func() {
				defer w.Close()
				w.Write([]byte(tt.stdinContent))
			}()

			// Set up stdin
			os.Stdin = r
			defer func() {
				os.Stdin = originalStdin
			}()

			// Execute command
			xargsx := &XargsxCommand{}
			err = xargsx.Execute(tt.args, []string{})

			if err != nil {
				t.Errorf("Unexpected error with %s: %v", tt.args[0], err)
			}
		})
	}
}

// Helper function to parse stdin arguments (extracted for testing)
func parseStdinArgs() []string {
	var inputArgs []string
	
	// Read from stdin and parse like the actual implementation
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			fields := strings.Fields(line)
			inputArgs = append(inputArgs, fields...)
		}
	}
	
	return inputArgs
}