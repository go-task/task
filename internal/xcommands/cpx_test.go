package xcommands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCpCommand_Execute(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Test cases
	tests := []struct {
		name        string
		args        []string
		flags       []string
		setup       func() error
		expectError bool
		validate    func() error
	}{
		{
			name:  "copy single file",
			args:  []string{filepath.Join(tempDir, "source.txt"), filepath.Join(tempDir, "dest.txt")},
			flags: []string{},
			setup: func() error {
				return os.WriteFile(filepath.Join(tempDir, "source.txt"), []byte("test content"), 0644)
			},
			expectError: false,
			validate: func() error {
				content, err := os.ReadFile(filepath.Join(tempDir, "dest.txt"))
				if err != nil {
					return err
				}
				if string(content) != "test content" {
					t.Errorf("Expected 'test content', got '%s'", string(content))
				}
				return nil
			},
		},
		{
			name:  "copy directory recursively",
			args:  []string{filepath.Join(tempDir, "sourcedir"), filepath.Join(tempDir, "destdir")},
			flags: []string{"r"},
			setup: func() error {
				sourceDir := filepath.Join(tempDir, "sourcedir")
				if err := os.MkdirAll(sourceDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("dir content"), 0644)
			},
			expectError: false,
			validate: func() error {
				content, err := os.ReadFile(filepath.Join(tempDir, "destdir", "file.txt"))
				if err != nil {
					return err
				}
				if string(content) != "dir content" {
					t.Errorf("Expected 'dir content', got '%s'", string(content))
				}
				return nil
			},
		},
		{
			name:        "missing operands",
			args:        []string{"single-arg"},
			flags:       []string{},
			setup:       func() error { return nil },
			expectError: true,
			validate:    func() error { return nil },
		},
		{
			name:  "copy directory without recursive flag",
			args:  []string{filepath.Join(tempDir, "sourcedir2"), filepath.Join(tempDir, "destdir2")},
			flags: []string{},
			setup: func() error {
				sourceDir := filepath.Join(tempDir, "sourcedir2")
				return os.MkdirAll(sourceDir, 0755)
			},
			expectError: true,
			validate:    func() error { return nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if err := tt.setup(); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Execute
			cp := &CpCommand{}
			err := cp.Execute(tt.args, tt.flags)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Validate result
			if !tt.expectError {
				if err := tt.validate(); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name          string
		input         []string
		expectedFlags []string
		expectedArgs  []string
	}{
		{
			name:          "no flags",
			input:         []string{"source.txt", "dest.txt"},
			expectedFlags: nil,
			expectedArgs:  []string{"source.txt", "dest.txt"},
		},
		{
			name:          "single dash flags",
			input:         []string{"-r", "-p", "source.txt", "dest.txt"},
			expectedFlags: []string{"r", "p"},
			expectedArgs:  []string{"source.txt", "dest.txt"},
		},
		{
			name:          "double dash flags",
			input:         []string{"--recursive", "--preserve", "source.txt", "dest.txt"},
			expectedFlags: []string{"recursive", "preserve"},
			expectedArgs:  []string{"source.txt", "dest.txt"},
		},
		{
			name:          "mixed flags and args",
			input:         []string{"-r", "source.txt", "-p", "dest.txt"},
			expectedFlags: []string{"r", "p"},
			expectedArgs:  []string{"source.txt", "dest.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, args := parseFlags(tt.input)

			// Check flags
			if len(flags) != len(tt.expectedFlags) {
				t.Errorf("Expected %d flags, got %d", len(tt.expectedFlags), len(flags))
			}
			for i, flag := range flags {
				if i >= len(tt.expectedFlags) || flag != tt.expectedFlags[i] {
					t.Errorf("Expected flag '%s', got '%s'", tt.expectedFlags[i], flag)
				}
			}

			// Check args
			if len(args) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(args))
			}
			for i, arg := range args {
				if i >= len(tt.expectedArgs) || arg != tt.expectedArgs[i] {
					t.Errorf("Expected arg '%s', got '%s'", tt.expectedArgs[i], arg)
				}
			}
		})
	}
}