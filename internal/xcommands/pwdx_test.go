package xcommands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPwdxCommand_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       []string
		expectError bool
		setup       func() (string, func(), error) // returns expected dir, cleanup func, error
	}{
		{
			name:        "print current working directory",
			args:        []string{},
			flags:       []string{},
			expectError: false,
			setup: func() (string, func(), error) {
				// No setup needed, just return current dir
				wd, err := os.Getwd()
				return wd, func() {}, err
			},
		},
		{
			name:        "ignore arguments",
			args:        []string{"ignored", "args"},
			flags:       []string{},
			expectError: false,
			setup: func() (string, func(), error) {
				wd, err := os.Getwd()
				return wd, func() {}, err
			},
		},
		{
			name:        "ignore flags",
			args:        []string{},
			flags:       []string{"ignored", "flags"},
			expectError: false,
			setup: func() (string, func(), error) {
				wd, err := os.Getwd()
				return wd, func() {}, err
			},
		},
		{
			name:        "works in different directory",
			args:        []string{},
			flags:       []string{},
			expectError: false,
			setup: func() (string, func(), error) {
				// Create temp dir and change to it
				tempDir := t.TempDir()
				originalDir, err := os.Getwd()
				if err != nil {
					return "", func() {}, err
				}
				
				err = os.Chdir(tempDir)
				if err != nil {
					return "", func() {}, err
				}
				
				cleanup := func() {
					os.Chdir(originalDir)
				}
				
				return tempDir, cleanup, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedDir, cleanup, err := tt.setup()
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			defer cleanup()

			// Capture stdout to verify output
			originalStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("Failed to create pipe: %v", err)
			}
			os.Stdout = w

			// Execute command
			pwd := &PwdxCommand{}
			execErr := pwd.Execute(tt.args, tt.flags)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = originalStdout
			
			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := strings.TrimSpace(string(buf[:n]))
			r.Close()

			// Check error expectation
			if tt.expectError && execErr == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && execErr != nil {
				t.Errorf("Unexpected error: %v", execErr)
			}

			// Verify output matches expected directory
			if !tt.expectError {
				// On macOS, temp directories may have symlinks resolved differently
				// Use filepath.EvalSymlinks to get the canonical path for comparison
				expectedCanonical, err := filepath.EvalSymlinks(expectedDir)
				if err != nil {
					expectedCanonical = expectedDir
				}
				outputCanonical, err := filepath.EvalSymlinks(output)
				if err != nil {
					outputCanonical = output
				}

				if outputCanonical != expectedCanonical {
					t.Errorf("Expected output '%s', got '%s'", expectedCanonical, outputCanonical)
				}

				// Also verify it's an absolute path
				if !filepath.IsAbs(output) {
					t.Errorf("Expected absolute path, got '%s'", output)
				}
			}
		})
	}
}

func TestPwdxCommand_Integration(t *testing.T) {
	// Test that pwdx command is properly registered
	mu.RLock()
	cmd, exists := registry["pwdx"]
	mu.RUnlock()

	if !exists {
		t.Error("pwdx command not registered")
		return
	}

	if _, ok := cmd.(*PwdxCommand); !ok {
		t.Error("pwdx command is not of correct type")
	}
}