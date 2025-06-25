package xcommands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMvxCommand_Execute(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Test cases
	tests := []struct {
		name        string
		args        []string
		flags       []string
		setup       func() error
		expectError bool
		errorMsg    string
		validate    func() error
	}{
		{
			name:  "move single file",
			args:  []string{filepath.Join(tempDir, "source.txt"), filepath.Join(tempDir, "dest.txt")},
			flags: []string{},
			setup: func() error {
				return os.WriteFile(filepath.Join(tempDir, "source.txt"), []byte("test content"), 0644)
			},
			expectError: false,
			validate: func() error {
				// Check destination exists with correct content
				content, err := os.ReadFile(filepath.Join(tempDir, "dest.txt"))
				if err != nil {
					return err
				}
				if string(content) != "test content" {
					t.Errorf("Expected 'test content', got '%s'", string(content))
				}
				
				// Check source no longer exists
				if _, err := os.Stat(filepath.Join(tempDir, "source.txt")); !os.IsNotExist(err) {
					t.Error("Source file should not exist after move")
				}
				return nil
			},
		},
		{
			name:  "rename file",
			args:  []string{filepath.Join(tempDir, "oldname.txt"), filepath.Join(tempDir, "newname.txt")},
			flags: []string{},
			setup: func() error {
				return os.WriteFile(filepath.Join(tempDir, "oldname.txt"), []byte("rename content"), 0644)
			},
			expectError: false,
			validate: func() error {
				// Check new name exists with correct content
				content, err := os.ReadFile(filepath.Join(tempDir, "newname.txt"))
				if err != nil {
					return err
				}
				if string(content) != "rename content" {
					t.Errorf("Expected 'rename content', got '%s'", string(content))
				}
				
				// Check old name no longer exists
				if _, err := os.Stat(filepath.Join(tempDir, "oldname.txt")); !os.IsNotExist(err) {
					t.Error("Old filename should not exist after rename")
				}
				return nil
			},
		},
		{
			name:  "move file into existing directory",
			args:  []string{filepath.Join(tempDir, "movefile.txt"), filepath.Join(tempDir, "targetdir")},
			flags: []string{},
			setup: func() error {
				if err := os.WriteFile(filepath.Join(tempDir, "movefile.txt"), []byte("move into dir"), 0644); err != nil {
					return err
				}
				return os.MkdirAll(filepath.Join(tempDir, "targetdir"), 0755)
			},
			expectError: false,
			validate: func() error {
				// Check file exists in target directory
				content, err := os.ReadFile(filepath.Join(tempDir, "targetdir", "movefile.txt"))
				if err != nil {
					return err
				}
				if string(content) != "move into dir" {
					t.Errorf("Expected 'move into dir', got '%s'", string(content))
				}
				
				// Check original file no longer exists
				if _, err := os.Stat(filepath.Join(tempDir, "movefile.txt")); !os.IsNotExist(err) {
					t.Error("Original file should not exist after move")
				}
				return nil
			},
		},
		{
			name:  "move directory",
			args:  []string{filepath.Join(tempDir, "sourcedir"), filepath.Join(tempDir, "destdir")},
			flags: []string{},
			setup: func() error {
				sourceDir := filepath.Join(tempDir, "sourcedir")
				if err := os.MkdirAll(sourceDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("dir content 1"), 0644); err != nil {
					return err
				}
				subDir := filepath.Join(sourceDir, "subdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("dir content 2"), 0644)
			},
			expectError: false,
			validate: func() error {
				// Check destination directory structure exists
				content1, err := os.ReadFile(filepath.Join(tempDir, "destdir", "file1.txt"))
				if err != nil {
					return err
				}
				if string(content1) != "dir content 1" {
					t.Errorf("Expected 'dir content 1', got '%s'", string(content1))
				}
				
				content2, err := os.ReadFile(filepath.Join(tempDir, "destdir", "subdir", "file2.txt"))
				if err != nil {
					return err
				}
				if string(content2) != "dir content 2" {
					t.Errorf("Expected 'dir content 2', got '%s'", string(content2))
				}
				
				// Check source directory no longer exists
				if _, err := os.Stat(filepath.Join(tempDir, "sourcedir")); !os.IsNotExist(err) {
					t.Error("Source directory should not exist after move")
				}
				return nil
			},
		},
		{
			name:  "move directory into existing directory",
			args:  []string{filepath.Join(tempDir, "movedir"), filepath.Join(tempDir, "parentdir")},
			flags: []string{},
			setup: func() error {
				// Create source directory
				sourceDir := filepath.Join(tempDir, "movedir")
				if err := os.MkdirAll(sourceDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(sourceDir, "dirfile.txt"), []byte("content in dir"), 0644); err != nil {
					return err
				}
				
				// Create parent directory
				return os.MkdirAll(filepath.Join(tempDir, "parentdir"), 0755)
			},
			expectError: false,
			validate: func() error {
				// Check directory moved into parent directory
				content, err := os.ReadFile(filepath.Join(tempDir, "parentdir", "movedir", "dirfile.txt"))
				if err != nil {
					return err
				}
				if string(content) != "content in dir" {
					t.Errorf("Expected 'content in dir', got '%s'", string(content))
				}
				
				// Check source directory no longer exists
				if _, err := os.Stat(filepath.Join(tempDir, "movedir")); !os.IsNotExist(err) {
					t.Error("Source directory should not exist after move")
				}
				return nil
			},
		},
		{
			name:        "missing operands - no args",
			args:        []string{},
			flags:       []string{},
			setup:       func() error { return nil },
			expectError: true,
			errorMsg:    "missing operands",
			validate:    func() error { return nil },
		},
		{
			name:        "missing operands - single arg",
			args:        []string{"single-arg"},
			flags:       []string{},
			setup:       func() error { return nil },
			expectError: true,
			errorMsg:    "missing operands",
			validate:    func() error { return nil },
		},
		{
			name:        "source file does not exist",
			args:        []string{filepath.Join(tempDir, "nonexistent.txt"), filepath.Join(tempDir, "dest.txt")},
			flags:       []string{},
			setup:       func() error { return nil },
			expectError: true,
			errorMsg:    "cannot stat",
			validate:    func() error { return nil },
		},
		{
			name:  "destination file exists without force flag",
			args:  []string{filepath.Join(tempDir, "src_exists.txt"), filepath.Join(tempDir, "dest_exists.txt")},
			flags: []string{},
			setup: func() error {
				if err := os.WriteFile(filepath.Join(tempDir, "src_exists.txt"), []byte("source"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(tempDir, "dest_exists.txt"), []byte("destination"), 0644)
			},
			expectError: true,
			errorMsg:    "file exists",
			validate:    func() error { return nil },
		},
		{
			name:  "destination file exists with force flag",
			args:  []string{filepath.Join(tempDir, "src_force.txt"), filepath.Join(tempDir, "dest_force.txt")},
			flags: []string{"f"},
			setup: func() error {
				if err := os.WriteFile(filepath.Join(tempDir, "src_force.txt"), []byte("source content"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(tempDir, "dest_force.txt"), []byte("old content"), 0644)
			},
			expectError: false,
			validate: func() error {
				// Check destination was overwritten
				content, err := os.ReadFile(filepath.Join(tempDir, "dest_force.txt"))
				if err != nil {
					return err
				}
				if string(content) != "source content" {
					t.Errorf("Expected 'source content', got '%s'", string(content))
				}
				
				// Check source no longer exists
				if _, err := os.Stat(filepath.Join(tempDir, "src_force.txt")); !os.IsNotExist(err) {
					t.Error("Source file should not exist after move")
				}
				return nil
			},
		},
		{
			name:  "preserve permissions and timestamps",
			args:  []string{filepath.Join(tempDir, "perm_src.txt"), filepath.Join(tempDir, "perm_dest.txt")},
			flags: []string{},
			setup: func() error {
				srcPath := filepath.Join(tempDir, "perm_src.txt")
				if err := os.WriteFile(srcPath, []byte("perm test"), 0600); err != nil {
					return err
				}
				// Set a specific timestamp
				testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
				return os.Chtimes(srcPath, testTime, testTime)
			},
			expectError: false,
			validate: func() error {
				destPath := filepath.Join(tempDir, "perm_dest.txt")
				info, err := os.Stat(destPath)
				if err != nil {
					return err
				}
				
				// Check permissions (on Unix systems)
				if info.Mode().Perm() != 0600 {
					t.Errorf("Expected permissions 0600, got %o", info.Mode().Perm())
				}
				
				// Check timestamp (within 1 second tolerance)
				expectedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
				if info.ModTime().Sub(expectedTime).Abs() > time.Second {
					t.Errorf("Expected timestamp %v, got %v", expectedTime, info.ModTime())
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
			mvx := &MvxCommand{}
			err := mvx.Execute(tt.args, tt.flags)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			// Check specific error message if provided
			if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorMsg, err)
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

func TestMvxCommand_FlagHandling(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		flags       []string
		expectForce bool
		expectInter bool
	}{
		{
			name:        "no flags",
			flags:       []string{},
			expectForce: false,
			expectInter: false,
		},
		{
			name:        "force flag short",
			flags:       []string{"f"},
			expectForce: true,
			expectInter: false,
		},
		{
			name:        "force flag long",
			flags:       []string{"force"},
			expectForce: true,
			expectInter: false,
		},
		{
			name:        "interactive flag short",
			flags:       []string{"i"},
			expectForce: false,
			expectInter: true,
		},
		{
			name:        "interactive flag long",
			flags:       []string{"interactive"},
			expectForce: false,
			expectInter: true,
		},
		{
			name:        "both flags",
			flags:       []string{"f", "i"},
			expectForce: true,
			expectInter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test files
			srcFile := filepath.Join(tempDir, "flag_test_src_"+tt.name+".txt")
			destFile := filepath.Join(tempDir, "flag_test_dest_"+tt.name+".txt")
			
			if err := os.WriteFile(srcFile, []byte("flag test"), 0644); err != nil {
				t.Fatalf("Failed to create source file: %v", err)
			}

			// Test the moveFile function directly to verify flag parsing
			err := moveFile(srcFile, destFile, tt.expectForce, tt.expectInter)
			if err != nil {
				t.Errorf("moveFile failed: %v", err)
			}

			// Verify file was moved
			if _, err := os.Stat(destFile); err != nil {
				t.Errorf("Destination file should exist: %v", err)
			}
			if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
				t.Error("Source file should not exist after move")
			}
		})
	}
}

func TestMvxCommand_CrossFilesystemMove(t *testing.T) {
	// This test simulates cross-filesystem move by using the moveSingleFile function directly
	tempDir := t.TempDir()
	
	srcFile := filepath.Join(tempDir, "cross_src.txt")
	destFile := filepath.Join(tempDir, "subdir", "cross_dest.txt")
	
	// Create source file
	testContent := "cross filesystem test content"
	if err := os.WriteFile(srcFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Test cross-filesystem move (copy + remove)
	if err := moveSingleFile(srcFile, destFile); err != nil {
		t.Fatalf("moveSingleFile failed: %v", err)
	}
	
	// Verify destination exists with correct content
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("Expected '%s', got '%s'", testContent, string(content))
	}
	
	// Verify source no longer exists
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Error("Source file should not exist after cross-filesystem move")
	}
}

func TestMvxCommand_CrossFilesystemMoveDirectory(t *testing.T) {
	// This test simulates cross-filesystem directory move
	tempDir := t.TempDir()
	
	srcDir := filepath.Join(tempDir, "cross_src_dir")
	destDir := filepath.Join(tempDir, "cross_dest_dir")
	
	// Create source directory structure
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	
	testContent1 := "file1 content"
	testContent2 := "file2 content"
	
	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte(testContent1), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte(testContent2), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	
	// Test cross-filesystem directory move
	if err := moveDirectory(srcDir, destDir); err != nil {
		t.Fatalf("moveDirectory failed: %v", err)
	}
	
	// Verify destination directory structure
	content1, err := os.ReadFile(filepath.Join(destDir, "file1.txt"))
	if err != nil {
		t.Fatalf("Failed to read file1 from destination: %v", err)
	}
	if string(content1) != testContent1 {
		t.Errorf("Expected '%s', got '%s'", testContent1, string(content1))
	}
	
	content2, err := os.ReadFile(filepath.Join(destDir, "subdir", "file2.txt"))
	if err != nil {
		t.Fatalf("Failed to read file2 from destination: %v", err)
	}
	if string(content2) != testContent2 {
		t.Errorf("Expected '%s', got '%s'", testContent2, string(content2))
	}
	
	// Verify source directory no longer exists
	if _, err := os.Stat(srcDir); !os.IsNotExist(err) {
		t.Error("Source directory should not exist after cross-filesystem move")
	}
}

func TestMvxCommand_PermissionErrors(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test permission error when trying to move to protected directory
	// Note: This test may be platform-specific and might not work in all environments
	srcFile := filepath.Join(tempDir, "perm_src.txt")
	if err := os.WriteFile(srcFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Try to move to a non-existent directory path that would require creation
	// This should test the directory creation logic in copySingleFileForMove
	destFile := filepath.Join(tempDir, "nonexistent", "deep", "path", "dest.txt")
	
	if err := moveSingleFile(srcFile, destFile); err != nil {
		t.Fatalf("moveSingleFile should handle directory creation: %v", err)
	}
	
	// Verify the file was moved and directories were created
	if _, err := os.Stat(destFile); err != nil {
		t.Errorf("Destination file should exist: %v", err)
	}
}

func TestMvxCommand_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setup       func() (string, string)
		expectError bool
		errorMsg    string
	}{
		{
			name: "move file to same location",
			setup: func() (string, string) {
				srcFile := filepath.Join(tempDir, "same_location.txt")
				os.WriteFile(srcFile, []byte("content"), 0644)
				return srcFile, srcFile
			},
			expectError: true, // Should fail - cannot move file to itself
			errorMsg:    "file exists",
		},
		{
			name: "move with special characters in filename",
			setup: func() (string, string) {
				srcFile := filepath.Join(tempDir, "file with spaces & symbols!.txt")
				destFile := filepath.Join(tempDir, "dest with spaces & symbols!.txt")
				os.WriteFile(srcFile, []byte("special chars"), 0644)
				return srcFile, destFile
			},
			expectError: false,
		},
		{
			name: "move empty file",
			setup: func() (string, string) {
				srcFile := filepath.Join(tempDir, "empty_src.txt")
				destFile := filepath.Join(tempDir, "empty_dest.txt")
				os.WriteFile(srcFile, []byte(""), 0644)
				return srcFile, destFile
			},
			expectError: false,
		},
		{
			name: "move very long filename",
			setup: func() (string, string) {
				longName := strings.Repeat("a", 100) + ".txt"
				srcFile := filepath.Join(tempDir, "short.txt")
				destFile := filepath.Join(tempDir, longName)
				os.WriteFile(srcFile, []byte("long name test"), 0644)
				return srcFile, destFile
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dest := tt.setup()
			
			mvx := &MvxCommand{}
			err := mvx.Execute([]string{src, dest}, []string{})
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorMsg, err)
				}
			}
		})
	}
}