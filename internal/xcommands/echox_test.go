package xcommands

import (
	"os"
	"strings"
	"testing"
)

func TestEchoxCommand_Execute(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		flags          []string
		expectedOutput string
		expectNewline  bool
	}{
		{
			name:           "simple echo",
			args:           []string{"hello"},
			flags:          []string{},
			expectedOutput: "hello",
			expectNewline:  true,
		},
		{
			name:           "echo multiple words",
			args:           []string{"hello", "world"},
			flags:          []string{},
			expectedOutput: "hello world",
			expectNewline:  true,
		},
		{
			name:           "echo with -n flag",
			args:           []string{"hello"},
			flags:          []string{"n"},
			expectedOutput: "hello",
			expectNewline:  false,
		},
		{
			name:           "echo multiple words with -n flag",
			args:           []string{"hello", "world"},
			flags:          []string{"n"},
			expectedOutput: "hello world",
			expectNewline:  false,
		},
		{
			name:           "echo empty string",
			args:           []string{},
			flags:          []string{},
			expectedOutput: "",
			expectNewline:  true,
		},
		{
			name:           "echo empty string with -n flag",
			args:           []string{},
			flags:          []string{"n"},
			expectedOutput: "",
			expectNewline:  false,
		},
		{
			name:           "echo with spaces",
			args:           []string{"hello", "", "world"},
			flags:          []string{},
			expectedOutput: "hello  world",
			expectNewline:  true,
		},
		{
			name:           "echo with special characters",
			args:           []string{"hello\tworld\n"},
			flags:          []string{},
			expectedOutput: "hello\tworld\n",
			expectNewline:  true,
		},
		{
			name:           "echo with unknown flag (ignored)",
			args:           []string{"hello"},
			flags:          []string{"unknown"},
			expectedOutput: "hello",
			expectNewline:  true,
		},
		{
			name:           "echo with -n and unknown flag",
			args:           []string{"hello"},
			flags:          []string{"n", "unknown"},
			expectedOutput: "hello",
			expectNewline:  false,
		},
		{
			name:           "echo with multiple same flags",
			args:           []string{"hello"},
			flags:          []string{"n", "n"},
			expectedOutput: "hello",
			expectNewline:  false,
		},
		{
			name:           "echo with numbers",
			args:           []string{"123", "456"},
			flags:          []string{},
			expectedOutput: "123 456",
			expectNewline:  true,
		},
		{
			name:           "echo with single quotes",
			args:           []string{"'hello world'"},
			flags:          []string{},
			expectedOutput: "'hello world'",
			expectNewline:  true,
		},
		{
			name:           "echo with double quotes",
			args:           []string{"\"hello world\""},
			flags:          []string{},
			expectedOutput: "\"hello world\"",
			expectNewline:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			originalStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			os.Stdout = w

			// Execute command
			echo := &EchoxCommand{}
			execErr := echo.Execute(tt.args, tt.flags)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = originalStdout

			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])
			r.Close()

			// Check that no error occurred
			if execErr != nil {
				t.Errorf("Unexpected error: %v", execErr)
			}

			// Verify output content
			if tt.expectNewline {
				expectedFull := tt.expectedOutput + "\n"
				if output != expectedFull {
					t.Errorf("Expected output '%s' (with newline), got '%s'", expectedFull, output)
				}
			} else {
				if output != tt.expectedOutput {
					t.Errorf("Expected output '%s' (without newline), got '%s'", tt.expectedOutput, output)
				}
			}

			// Verify newline handling
			hasNewline := strings.HasSuffix(output, "\n")
			if tt.expectNewline && !hasNewline {
				t.Error("Expected newline at end of output but didn't find one")
			}
			if !tt.expectNewline && hasNewline {
				t.Error("Expected no newline at end of output but found one")
			}
		})
	}
}

func TestEchoxCommand_FlagParsing(t *testing.T) {
	tests := []struct {
		name           string
		flags          []string
		expectedNoNewline bool
	}{
		{
			name:           "no flags",
			flags:          []string{},
			expectedNoNewline: false,
		},
		{
			name:           "n flag",
			flags:          []string{"n"},
			expectedNoNewline: true,
		},
		{
			name:           "multiple flags including n",
			flags:          []string{"x", "n", "y"},
			expectedNoNewline: true,
		},
		{
			name:           "unknown flags only",
			flags:          []string{"x", "y", "z"},
			expectedNoNewline: false,
		},
		{
			name:           "duplicate n flags",
			flags:          []string{"n", "n"},
			expectedNoNewline: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			originalStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			os.Stdout = w

			// Execute command with test string
			echo := &EchoxCommand{}
			execErr := echo.Execute([]string{"test"}, tt.flags)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = originalStdout

			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])
			r.Close()

			// Check that no error occurred
			if execErr != nil {
				t.Errorf("Unexpected error: %v", execErr)
			}

			// Verify newline behavior
			hasNewline := strings.HasSuffix(output, "\n")
			if tt.expectedNoNewline && hasNewline {
				t.Error("Expected no newline but found one")
			}
			if !tt.expectedNoNewline && !hasNewline {
				t.Error("Expected newline but didn't find one")
			}
		})
	}
}

func TestEchoxCommand_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		flags    []string
		expected string
	}{
		{
			name:     "echo with only spaces",
			args:     []string{" ", " "},
			flags:    []string{},
			expected: "   \n", // Two spaces joined with one space between them
		},
		{
			name:     "echo with empty string argument",
			args:     []string{""},
			flags:    []string{},
			expected: "\n",
		},
		{
			name:     "echo with mixed empty and non-empty",
			args:     []string{"hello", "", "world"},
			flags:    []string{},
			expected: "hello  world\n",
		},
		{
			name:     "echo with tab character",
			args:     []string{"hello\tworld"},
			flags:    []string{},
			expected: "hello\tworld\n",
		},
		{
			name:     "echo with newline character in argument",
			args:     []string{"hello\nworld"},
			flags:    []string{},
			expected: "hello\nworld\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			originalStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			os.Stdout = w

			// Execute command
			echo := &EchoxCommand{}
			execErr := echo.Execute(tt.args, tt.flags)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = originalStdout

			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])
			r.Close()

			// Check that no error occurred
			if execErr != nil {
				t.Errorf("Unexpected error: %v", execErr)
			}

			// Verify output
			if output != tt.expected {
				t.Errorf("Expected output '%q', got '%q'", tt.expected, output)
			}
		})
	}
}

func TestEchoxCommand_Integration(t *testing.T) {
	// Test that echox command is properly registered
	mu.RLock()
	cmd, exists := registry["echox"]
	mu.RUnlock()

	if !exists {
		t.Error("echox command not registered")
		return
	}

	if _, ok := cmd.(*EchoxCommand); !ok {
		t.Error("echox command is not of correct type")
	}
}

func TestEchoxCommand_WithParseFlags(t *testing.T) {
	// Test that echox works correctly with the parseFlags function
	tests := []struct {
		name           string
		rawArgs        []string
		expectedOutput string
		expectNewline  bool
	}{
		{
			name:           "echo with -n flag parsed",
			rawArgs:        []string{"-n", "hello", "world"},
			expectedOutput: "hello world",
			expectNewline:  false,
		},
		{
			name:           "echo with --n flag parsed",
			rawArgs:        []string{"--n", "hello", "world"},
			expectedOutput: "hello world",
			expectNewline:  false,
		},
		{
			name:           "echo with mixed args and flags",
			rawArgs:        []string{"hello", "-n", "world"},
			expectedOutput: "hello world",
			expectNewline:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the same parseFlags function that the coreutils system uses
			flags, args := parseFlags(tt.rawArgs)

			// Capture stdout
			originalStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			os.Stdout = w

			// Execute command
			echo := &EchoxCommand{}
			execErr := echo.Execute(args, flags)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = originalStdout

			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])
			r.Close()

			// Check that no error occurred
			if execErr != nil {
				t.Errorf("Unexpected error: %v", execErr)
			}

			// Verify output content
			if tt.expectNewline {
				expectedFull := tt.expectedOutput + "\n"
				if output != expectedFull {
					t.Errorf("Expected output '%s' (with newline), got '%s'", expectedFull, output)
				}
			} else {
				if output != tt.expectedOutput {
					t.Errorf("Expected output '%s' (without newline), got '%s'", tt.expectedOutput, output)
				}
			}
		})
	}
}