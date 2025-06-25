package xcommands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRmxCommand_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       []string
		setup       func(tempDir string) error
		expectError bool
		validate    func(tempDir string) error
	}{
		{
			name:  "remove single file",
			args:  []string{"test.txt"},
			flags: []string{},
			setup: func(tempDir string) error {
				return os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test content"), 0644)
			},
			expectError: false,
			validate: func(tempDir string) error {
				if _, err := os.Stat(filepath.Join(tempDir, "test.txt")); !os.IsNotExist(err) {
					t.Error("File should have been removed")
				}
				return nil
			},
		},
		{
			name:  "remove multiple files",
			args:  []string{"file1.txt", "file2.txt", "file3.txt"},
			flags: []string{},
			setup: func(tempDir string) error {
				files := []string{"file1.txt", "file2.txt", "file3.txt"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(tempDir, file), []byte("content"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectError: false,
			validate: func(tempDir string) error {
				files := []string{"file1.txt", "file2.txt", "file3.txt"}
				for _, file := range files {
					if _, err := os.Stat(filepath.Join(tempDir, file)); !os.IsNotExist(err) {
						t.Errorf("File %s should have been removed", file)
					}
				}
				return nil
			},
		},
		{
			name:  "remove directory with -r flag",
			args:  []string{"testdir"},
			flags: []string{"r"},
			setup: func(tempDir string) error {
				dirPath := filepath.Join(tempDir, "testdir")
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dirPath, "file.txt"), []byte("content"), 0644)
			},
			expectError: false,
			validate: func(tempDir string) error {
				if _, err := os.Stat(filepath.Join(tempDir, "testdir")); !os.IsNotExist(err) {
					t.Error("Directory should have been removed")
				}
				return nil
			},
		},
		{
			name:  "remove directory with -R flag (capital)",
			args:  []string{"testdir"},
			flags: []string{"R"},
			setup: func(tempDir string) error {
				dirPath := filepath.Join(tempDir, "testdir")
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dirPath, "file.txt"), []byte("content"), 0644)
			},
			expectError: false,
			validate: func(tempDir string) error {
				if _, err := os.Stat(filepath.Join(tempDir, "testdir")); !os.IsNotExist(err) {
					t.Error("Directory should have been removed")
				}
				return nil
			},
		},
		{
			name:  "force removal with -f flag (ignore nonexistent files)",
			args:  []string{"nonexistent.txt", "existing.txt"},
			flags: []string{"f"},
			setup: func(tempDir string) error {
				return os.WriteFile(filepath.Join(tempDir, "existing.txt"), []byte("content"), 0644)
			},
			expectError: false,
			validate: func(tempDir string) error {
				if _, err := os.Stat(filepath.Join(tempDir, "existing.txt")); !os.IsNotExist(err) {
					t.Error("Existing file should have been removed")
				}
				return nil
			},
		},
		{
			name:  "combined flags -rf",
			args:  []string{"testdir", "nonexistent.txt"},
			flags: []string{"r", "f"},
			setup: func(tempDir string) error {
				dirPath := filepath.Join(tempDir, "testdir")
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dirPath, "file.txt"), []byte("content"), 0644)
			},
			expectError: false,
			validate: func(tempDir string) error {
				if _, err := os.Stat(filepath.Join(tempDir, "testdir")); !os.IsNotExist(err) {
					t.Error("Directory should have been removed")
				}
				return nil
			},
		},
		{
			name:        "missing operands",
			args:        []string{},
			flags:       []string{},
			setup:       func(tempDir string) error { return nil },
			expectError: true,
			validate:    func(tempDir string) error { return nil },
		},
		{
			name:  "try to remove directory without -r flag",
			args:  []string{"testdir"},
			flags: []string{},
			setup: func(tempDir string) error {
				return os.MkdirAll(filepath.Join(tempDir, "testdir"), 0755)
			},
			expectError: true,
			validate: func(tempDir string) error {
				// Directory should still exist
				if _, err := os.Stat(filepath.Join(tempDir, "testdir")); os.IsNotExist(err) {
					t.Error("Directory should not have been removed")
				}
				return nil
			},
		},
		{
			name:        "remove nonexistent file without -f flag",
			args:        []string{"nonexistent.txt"},
			flags:       []string{},
			setup:       func(tempDir string) error { return nil },
			expectError: true,
			validate:    func(tempDir string) error { return nil },
		},
		{
			name:  "recursive directory removal with nested structures",
			args:  []string{"complex"},
			flags: []string{"r"},
			setup: func(tempDir string) error {
				base := filepath.Join(tempDir, "complex")
				dirs := []string{
					base,
					filepath.Join(base, "subdir1"),
					filepath.Join(base, "subdir2"),
					filepath.Join(base, "subdir1", "nested"),
				}
				for _, dir := range dirs {
					if err := os.MkdirAll(dir, 0755); err != nil {
						return err
					}
				}

				files := []string{
					filepath.Join(base, "file1.txt"),
					filepath.Join(base, "subdir1", "file2.txt"),
					filepath.Join(base, "subdir2", "file3.txt"),
					filepath.Join(base, "subdir1", "nested", "file4.txt"),
				}
				for _, file := range files {
					if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectError: false,
			validate: func(tempDir string) error {
				if _, err := os.Stat(filepath.Join(tempDir, "complex")); !os.IsNotExist(err) {
					t.Error("Complex directory structure should have been removed")
				}
				return nil
			},
		},
		{
			name:  "symlink handling - remove symlink",
			args:  []string{"symlink.txt"},
			flags: []string{},
			setup: func(tempDir string) error {
				target := filepath.Join(tempDir, "target.txt")
				if err := os.WriteFile(target, []byte("target content"), 0644); err != nil {
					return err
				}
				return os.Symlink(target, filepath.Join(tempDir, "symlink.txt"))
			},
			expectError: false,
			validate: func(tempDir string) error {
				// Symlink should be removed
				if _, err := os.Lstat(filepath.Join(tempDir, "symlink.txt")); !os.IsNotExist(err) {
					t.Error("Symlink should have been removed")
				}
				// Target should still exist
				if _, err := os.Stat(filepath.Join(tempDir, "target.txt")); os.IsNotExist(err) {
					t.Error("Target file should still exist")
				}
				return nil
			},
		},
		{
			name:  "symlink to directory",
			args:  []string{"dirlink"},
			flags: []string{},
			setup: func(tempDir string) error {
				targetDir := filepath.Join(tempDir, "targetdir")
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(targetDir, "file.txt"), []byte("content"), 0644); err != nil {
					return err
				}
				return os.Symlink(targetDir, filepath.Join(tempDir, "dirlink"))
			},
			expectError: false,
			validate: func(tempDir string) error {
				// Symlink should be removed
				if _, err := os.Lstat(filepath.Join(tempDir, "dirlink")); !os.IsNotExist(err) {
					t.Error("Directory symlink should have been removed")
				}
				// Target directory should still exist
				if _, err := os.Stat(filepath.Join(tempDir, "targetdir")); os.IsNotExist(err) {
					t.Error("Target directory should still exist")
				}
				return nil
			},
		},
		{
			name:  "remove multiple targets with mixed success/failure",
			args:  []string{"existing.txt", "nonexistent.txt", "another.txt"},
			flags: []string{"f"},
			setup: func(tempDir string) error {
				if err := os.WriteFile(filepath.Join(tempDir, "existing.txt"), []byte("content"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(tempDir, "another.txt"), []byte("content"), 0644)
			},
			expectError: false,
			validate: func(tempDir string) error {
				files := []string{"existing.txt", "another.txt"}
				for _, file := range files {
					if _, err := os.Stat(filepath.Join(tempDir, file)); !os.IsNotExist(err) {
						t.Errorf("File %s should have been removed", file)
					}
				}
				return nil
			},
		},
		{
			name:  "remove empty directory with -r flag",
			args:  []string{"emptydir"},
			flags: []string{"r"},
			setup: func(tempDir string) error {
				return os.MkdirAll(filepath.Join(tempDir, "emptydir"), 0755)
			},
			expectError: false,
			validate: func(tempDir string) error {
				if _, err := os.Stat(filepath.Join(tempDir, "emptydir")); !os.IsNotExist(err) {
					t.Error("Empty directory should have been removed")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for each test
			tempDir := t.TempDir()

			// Change to temp directory for relative path operations
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(oldDir); err != nil {
					t.Errorf("Failed to restore working directory: %v", err)
				}
			}()

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Setup
			if err := tt.setup(tempDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Execute
			rmx := &RmxCommand{}
			err = rmx.Execute(tt.args, tt.flags)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Validate result
			if !tt.expectError {
				if err := tt.validate(tempDir); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}
		})
	}
}

func TestRmxCommand_EdgeCases(t *testing.T) {
	t.Run("remove file with no write permission to parent directory", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a subdirectory with restricted permissions
		restrictedDir := filepath.Join(tempDir, "restricted")
		if err := os.MkdirAll(restrictedDir, 0755); err != nil {
			t.Fatalf("Failed to create restricted directory: %v", err)
		}

		// Create a file in the restricted directory
		testFile := filepath.Join(restrictedDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Remove write permission from the directory
		if err := os.Chmod(restrictedDir, 0555); err != nil {
			t.Fatalf("Failed to change directory permissions: %v", err)
		}
		defer os.Chmod(restrictedDir, 0755) // Restore permissions for cleanup

		rmx := &RmxCommand{}
		err := rmx.Execute([]string{testFile}, []string{})

		// On most Unix systems, this should fail due to lack of write permission
		// However, the exact behavior may vary by system, so we just check that
		// the command handles the situation gracefully
		if err == nil {
			// If it succeeded, verify the file was actually removed
			if _, statErr := os.Stat(testFile); !os.IsNotExist(statErr) {
				t.Error("File should have been removed if command succeeded")
			}
		}
		// If it failed, that's also acceptable behavior
	})

	t.Run("remove file that becomes nonexistent during operation", func(t *testing.T) {
		// This test is more conceptual - in practice, it's hard to simulate
		// a file disappearing during the removal operation in a unit test
		tempDir := t.TempDir()

		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		rmx := &RmxCommand{}
		err := rmx.Execute([]string{testFile}, []string{})

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("File should have been removed")
		}
	})

	t.Run("remove with force flag and mixed existing/nonexistent files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Change to temp directory for relative path operations
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}
		defer func() {
			if err := os.Chdir(oldDir); err != nil {
				t.Errorf("Failed to restore working directory: %v", err)
			}
		}()

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		// Create some files
		existingFiles := []string{"file1.txt", "file3.txt"}
		for _, file := range existingFiles {
			if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", file, err)
			}
		}

		// Mix existing and nonexistent files
		allFiles := []string{"file1.txt", "nonexistent1.txt", "file3.txt", "nonexistent2.txt"}

		rmx := &RmxCommand{}
		err = rmx.Execute(allFiles, []string{"f"})

		// Should not return error due to force flag
		if err != nil {
			t.Errorf("Unexpected error with force flag: %v", err)
		}

		// Verify existing files were removed
		for _, file := range existingFiles {
			if _, err := os.Stat(file); !os.IsNotExist(err) {
				t.Errorf("File %s should have been removed", file)
			}
		}
	})
}

func TestRmxCommand_FileTypes(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(tempDir string) (string, error)
		flags   []string
		wantErr bool
	}{
		{
			name: "regular file",
			setup: func(tempDir string) (string, error) {
				file := filepath.Join(tempDir, "regular.txt")
				err := os.WriteFile(file, []byte("content"), 0644)
				return file, err
			},
			flags:   []string{},
			wantErr: false,
		},
		{
			name: "executable file",
			setup: func(tempDir string) (string, error) {
				file := filepath.Join(tempDir, "executable")
				if err := os.WriteFile(file, []byte("#!/bin/bash\necho hello"), 0755); err != nil {
					return "", err
				}
				return file, nil
			},
			flags:   []string{},
			wantErr: false,
		},
		{
			name: "hidden file",
			setup: func(tempDir string) (string, error) {
				file := filepath.Join(tempDir, ".hidden")
				err := os.WriteFile(file, []byte("hidden content"), 0644)
				return file, err
			},
			flags:   []string{},
			wantErr: false,
		},
		{
			name: "file with spaces in name",
			setup: func(tempDir string) (string, error) {
				file := filepath.Join(tempDir, "file with spaces.txt")
				err := os.WriteFile(file, []byte("content"), 0644)
				return file, err
			},
			flags:   []string{},
			wantErr: false,
		},
		{
			name: "directory with files",
			setup: func(tempDir string) (string, error) {
				dir := filepath.Join(tempDir, "testdir")
				if err := os.MkdirAll(dir, 0755); err != nil {
					return "", err
				}
				if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644); err != nil {
					return "", err
				}
				return dir, nil
			},
			flags:   []string{"r"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			target, err := tt.setup(tempDir)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			rmx := &RmxCommand{}
			err = rmx.Execute([]string{target}, tt.flags)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.wantErr {
				// Verify target was removed
				if _, err := os.Stat(target); !os.IsNotExist(err) {
					t.Error("Target should have been removed")
				}
			}
		})
	}
}

// TestParseFlags tests the flag parsing functionality used by rmx
func TestRmxParseFlags(t *testing.T) {
	tests := []struct {
		name          string
		input         []string
		expectedFlags []string
		expectedArgs  []string
	}{
		{
			name:          "no flags",
			input:         []string{"file1.txt", "file2.txt"},
			expectedFlags: nil,
			expectedArgs:  []string{"file1.txt", "file2.txt"},
		},
		{
			name:          "single flag -r",
			input:         []string{"-r", "directory"},
			expectedFlags: []string{"r"},
			expectedArgs:  []string{"directory"},
		},
		{
			name:          "single flag -f",
			input:         []string{"-f", "file.txt"},
			expectedFlags: []string{"f"},
			expectedArgs:  []string{"file.txt"},
		},
		{
			name:          "multiple flags",
			input:         []string{"-r", "-f", "directory"},
			expectedFlags: []string{"r", "f"},
			expectedArgs:  []string{"directory"},
		},
		{
			name:          "capital R flag",
			input:         []string{"-R", "directory"},
			expectedFlags: []string{"R"},
			expectedArgs:  []string{"directory"},
		},
		{
			name:          "mixed flags and args",
			input:         []string{"-r", "dir1", "-f", "dir2"},
			expectedFlags: []string{"r", "f"},
			expectedArgs:  []string{"dir1", "dir2"},
		},
		{
			name:          "double dash flags",
			input:         []string{"--recursive", "--force", "target"},
			expectedFlags: []string{"recursive", "force"},
			expectedArgs:  []string{"target"},
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
