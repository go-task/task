package xcommands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMkdirxCommand_Execute(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		args        []string
		flags       []string
		setup       func() error
		expectError bool
		errorContains string
		validate    func() error
	}{
		{
			name:        "create single directory",
			args:        []string{filepath.Join(tempDir, "testdir")},
			flags:       []string{},
			setup:       func() error { return nil },
			expectError: false,
			validate: func() error {
				info, err := os.Stat(filepath.Join(tempDir, "testdir"))
				if err != nil {
					return err
				}
				if !info.IsDir() {
					t.Error("Expected directory to be created")
				}
				if info.Mode().Perm() != 0755 {
					t.Errorf("Expected permissions 0755, got %o", info.Mode().Perm())
				}
				return nil
			},
		},
		{
			name:        "create multiple directories",
			args:        []string{filepath.Join(tempDir, "dir1"), filepath.Join(tempDir, "dir2"), filepath.Join(tempDir, "dir3")},
			flags:       []string{},
			setup:       func() error { return nil },
			expectError: false,
			validate: func() error {
				for _, dir := range []string{"dir1", "dir2", "dir3"} {
					info, err := os.Stat(filepath.Join(tempDir, dir))
					if err != nil {
						return err
					}
					if !info.IsDir() {
						t.Errorf("Expected directory %s to be created", dir)
					}
				}
				return nil
			},
		},
		{
			name:        "create nested directories with -p flag",
			args:        []string{filepath.Join(tempDir, "nested", "deep", "structure")},
			flags:       []string{"p"},
			setup:       func() error { return nil },
			expectError: false,
			validate: func() error {
				info, err := os.Stat(filepath.Join(tempDir, "nested", "deep", "structure"))
				if err != nil {
					return err
				}
				if !info.IsDir() {
					t.Error("Expected nested directory to be created")
				}
				return nil
			},
		},
		{
			name:        "create deeply nested path with -p flag",
			args:        []string{filepath.Join(tempDir, "very", "deep", "nested", "directory", "structure", "here")},
			flags:       []string{"p"},
			setup:       func() error { return nil },
			expectError: false,
			validate: func() error {
				info, err := os.Stat(filepath.Join(tempDir, "very", "deep", "nested", "directory", "structure", "here"))
				if err != nil {
					return err
				}
				if !info.IsDir() {
					t.Error("Expected deeply nested directory to be created")
				}
				return nil
			},
		},
		{
			name:        "create multiple nested directories with -p flag",
			args:        []string{filepath.Join(tempDir, "multi1", "sub1"), filepath.Join(tempDir, "multi2", "sub2", "subsub")},
			flags:       []string{"p"},
			setup:       func() error { return nil },
			expectError: false,
			validate: func() error {
				paths := []string{
					filepath.Join(tempDir, "multi1", "sub1"),
					filepath.Join(tempDir, "multi2", "sub2", "subsub"),
				}
				for _, path := range paths {
					info, err := os.Stat(path)
					if err != nil {
						return err
					}
					if !info.IsDir() {
						t.Errorf("Expected directory %s to be created", path)
					}
				}
				return nil
			},
		},
		{
			name:        "create directory with special characters",
			args:        []string{filepath.Join(tempDir, "special-chars_123", "sub dir with spaces")},
			flags:       []string{"p"},
			setup:       func() error { return nil },
			expectError: false,
			validate: func() error {
				info, err := os.Stat(filepath.Join(tempDir, "special-chars_123", "sub dir with spaces"))
				if err != nil {
					return err
				}
				if !info.IsDir() {
					t.Error("Expected directory with special characters to be created")
				}
				return nil
			},
		},
		{
			name:          "missing operands",
			args:          []string{},
			flags:         []string{},
			setup:         func() error { return nil },
			expectError:   true,
			errorContains: "missing operands",
			validate:      func() error { return nil },
		},
		{
			name:          "parent doesn't exist without -p flag",
			args:          []string{filepath.Join(tempDir, "nonexistent", "subdir")},
			flags:         []string{},
			setup:         func() error { return nil },
			expectError:   true,
			errorContains: "No such file or directory",
			validate:      func() error { return nil },
		},
		{
			name:        "directory already exists without -p flag",
			args:        []string{filepath.Join(tempDir, "existing")},
			flags:       []string{},
			setup: func() error {
				return os.Mkdir(filepath.Join(tempDir, "existing"), 0755)
			},
			expectError:   true,
			errorContains: "File exists",
			validate:      func() error { return nil },
		},
		{
			name:        "directory already exists with -p flag (should succeed)",
			args:        []string{filepath.Join(tempDir, "existing-p")},
			flags:       []string{"p"},
			setup: func() error {
				return os.Mkdir(filepath.Join(tempDir, "existing-p"), 0755)
			},
			expectError: false,
			validate: func() error {
				info, err := os.Stat(filepath.Join(tempDir, "existing-p"))
				if err != nil {
					return err
				}
				if !info.IsDir() {
					t.Error("Expected existing directory to remain")
				}
				return nil
			},
		},
		{
			name:        "nested directory already exists with -p flag",
			args:        []string{filepath.Join(tempDir, "existing-nested", "sub", "deep")},
			flags:       []string{"p"},
			setup: func() error {
				return os.MkdirAll(filepath.Join(tempDir, "existing-nested", "sub"), 0755)
			},
			expectError: false,
			validate: func() error {
				info, err := os.Stat(filepath.Join(tempDir, "existing-nested", "sub", "deep"))
				if err != nil {
					return err
				}
				if !info.IsDir() {
					t.Error("Expected nested directory to be created")
				}
				return nil
			},
		},
		{
			name:          "invalid flag",
			args:          []string{filepath.Join(tempDir, "flagtest")},
			flags:         []string{"invalid"},
			setup:         func() error { return nil },
			expectError:   true,
			errorContains: "invalid option",
			validate:      func() error { return nil },
		},
		{
			name:        "create directory on file (should fail)",
			args:        []string{filepath.Join(tempDir, "conflictfile", "subdir")},
			flags:       []string{},
			setup: func() error {
				return os.WriteFile(filepath.Join(tempDir, "conflictfile"), []byte("content"), 0644)
			},
			expectError:   true,
			errorContains: "not a directory",
			validate:      func() error { return nil },
		},
		{
			name:        "create directory on file with -p flag (should fail)",
			args:        []string{filepath.Join(tempDir, "conflictfile-p", "subdir")},
			flags:       []string{"p"},
			setup: func() error {
				return os.WriteFile(filepath.Join(tempDir, "conflictfile-p"), []byte("content"), 0644)
			},
			expectError:   true,
			errorContains: "not a directory",
			validate:      func() error { return nil },
		},
		{
			name:        "create directory with empty string (edge case)",
			args:        []string{""},
			flags:       []string{},
			setup:       func() error { return nil },
			expectError: true,
			validate:    func() error { return nil },
		},
		{
			name:        "create directory with dot (current directory)",
			args:        []string{filepath.Join(tempDir, ".")},
			flags:       []string{},
			setup:       func() error { return nil },
			expectError: true,
			errorContains: "File exists",
			validate:      func() error { return nil },
		},
		{
			name:        "create directory with dot and -p flag (should succeed)",
			args:        []string{filepath.Join(tempDir, ".")},
			flags:       []string{"p"},
			setup:       func() error { return nil },
			expectError: false,
			validate: func() error {
				// Dot directory should exist (it's the tempDir itself)
				info, err := os.Stat(tempDir)
				if err != nil {
					return err
				}
				if !info.IsDir() {
					t.Error("Expected current directory to exist")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if err := tt.setup(); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Execute
			mkdir := &MkdirxCommand{}
			err := mkdir.Execute(tt.args, tt.flags)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check error message content if specified
			if tt.expectError && err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
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

func TestMkdirxCommand_FlagHandling(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		flags       []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid -p flag",
			flags:       []string{"p"},
			expectError: false,
		},
		{
			name:        "invalid flag",
			flags:       []string{"x"},
			expectError: true,
			errorMsg:    "invalid option -- 'x'",
		},
		{
			name:        "multiple invalid flags",
			flags:       []string{"p", "invalid", "another"},
			expectError: true,
			errorMsg:    "invalid option -- 'invalid'",
		},
		{
			name:        "empty flag (edge case)",
			flags:       []string{""},
			expectError: true,
			errorMsg:    "invalid option -- ''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mkdir := &MkdirxCommand{}
			err := mkdir.Execute([]string{filepath.Join(tempDir, "flagtest")}, tt.flags)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestMkdirxCommand_PermissionHandling(t *testing.T) {
	tempDir := t.TempDir()

	// Test that directories are created with correct permissions (0755)
	mkdir := &MkdirxCommand{}
	testDir := filepath.Join(tempDir, "permtest")
	
	err := mkdir.Execute([]string{testDir}, []string{})
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	info, err := os.Stat(testDir)
	if err != nil {
		t.Fatalf("Failed to stat created directory: %v", err)
	}

	expectedPerm := os.FileMode(0755)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("Expected permissions %o, got %o", expectedPerm, info.Mode().Perm())
	}
}

func TestMkdirxCommand_Integration(t *testing.T) {
	tempDir := t.TempDir()

	// Test a complex real-world scenario
	mkdir := &MkdirxCommand{}

	// Create a complex directory structure
	paths := []string{
		filepath.Join(tempDir, "project", "src", "main"),
		filepath.Join(tempDir, "project", "src", "test"),
		filepath.Join(tempDir, "project", "docs", "api"),
		filepath.Join(tempDir, "project", "build", "output"),
	}

	err := mkdir.Execute(paths, []string{"p"})
	if err != nil {
		t.Fatalf("Failed to create complex directory structure: %v", err)
	}

	// Verify all paths were created
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Failed to stat directory %s: %v", path, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("Expected %s to be a directory", path)
		}
	}

	// Verify intermediate directories were also created
	intermediatePaths := []string{
		filepath.Join(tempDir, "project"),
		filepath.Join(tempDir, "project", "src"),
		filepath.Join(tempDir, "project", "docs"),
		filepath.Join(tempDir, "project", "build"),
	}

	for _, path := range intermediatePaths {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Failed to stat intermediate directory %s: %v", path, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("Expected intermediate path %s to be a directory", path)
		}
	}
}

func TestCreateDirectory(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		dir           string
		createParents bool
		setup         func() error
		expectError   bool
		errorContains string
	}{
		{
			name:          "create simple directory without parents",
			dir:           filepath.Join(tempDir, "simple"),
			createParents: false,
			setup:         func() error { return nil },
			expectError:   false,
		},
		{
			name:          "create nested directory with parents",
			dir:           filepath.Join(tempDir, "nested", "deep"),
			createParents: true,
			setup:         func() error { return nil },
			expectError:   false,
		},
		{
			name:          "create nested directory without parents (should fail)",
			dir:           filepath.Join(tempDir, "nested2", "deep2"),
			createParents: false,
			setup:         func() error { return nil },
			expectError:   true,
			errorContains: "No such file or directory",
		},
		{
			name:          "create directory that already exists without parents",
			dir:           filepath.Join(tempDir, "existing"),
			createParents: false,
			setup: func() error {
				return os.Mkdir(filepath.Join(tempDir, "existing"), 0755)
			},
			expectError:   true,
			errorContains: "File exists",
		},
		{
			name:          "create directory that already exists with parents",
			dir:           filepath.Join(tempDir, "existing2"),
			createParents: true,
			setup: func() error {
				return os.Mkdir(filepath.Join(tempDir, "existing2"), 0755)
			},
			expectError: false,
		},
		{
			name:          "create directory with MkdirAll error path",
			dir:           filepath.Join(tempDir, "fileconflict", "subdir"),
			createParents: true,
			setup: func() error {
				// Create a file where we want to create a directory
				return os.WriteFile(filepath.Join(tempDir, "fileconflict"), []byte("content"), 0644)
			},
			expectError:   true,
			errorContains: "cannot create directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if err := tt.setup(); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Execute
			err := createDirectory(tt.dir, tt.createParents)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check error message content if specified
			if tt.expectError && err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			}

			// Validate directory was created if no error expected
			if !tt.expectError {
				info, err := os.Stat(tt.dir)
				if err != nil {
					t.Errorf("Failed to stat created directory: %v", err)
				} else if !info.IsDir() {
					t.Error("Expected created path to be a directory")
				}
			}
		})
	}
}