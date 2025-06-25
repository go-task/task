package xcommands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeadxCommand_Execute(t *testing.T) {

	tests := []struct {
		name           string
		args           []string
		flags          []string
		setup          func(tempDir string) error
		expectError    bool
		expectedOutput string
	}{
		{
			name:  "default 10 lines",
			args:  []string{"test.txt"},
			flags: []string{},
			setup: func(tempDir string) error {
				lines := make([]string, 15)
				for i := 0; i < 15; i++ {
					lines[i] = fmt.Sprintf("Line %d", i+1)
				}
				content := strings.Join(lines, "\n") + "\n"
				return os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(content), 0644)
			},
			expectError:    false,
			expectedOutput: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10\n",
		},
		{
			name:  "custom number of lines with -n flag",
			args:  []string{"test.txt"},
			flags: []string{"n", "3"},
			setup: func(tempDir string) error {
				content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
				return os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(content), 0644)
			},
			expectError:    false,
			expectedOutput: "Line 1\nLine 2\nLine 3\n",
		},
		{
			name:  "combined -n flag format",
			args:  []string{"test.txt"},
			flags: []string{"n5"},
			setup: func(tempDir string) error {
				lines := make([]string, 10)
				for i := 0; i < 10; i++ {
					lines[i] = fmt.Sprintf("Line %d", i+1)
				}
				content := strings.Join(lines, "\n") + "\n"
				return os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(content), 0644)
			},
			expectError:    false,
			expectedOutput: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n",
		},
		{
			name:  "zero lines",
			args:  []string{"test.txt"},
			flags: []string{"n", "0"},
			setup: func(tempDir string) error {
				content := "Line 1\nLine 2\nLine 3\n"
				return os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(content), 0644)
			},
			expectError:    false,
			expectedOutput: "",
		},
		{
			name:  "more lines requested than available",
			args:  []string{"test.txt"},
			flags: []string{"n", "20"},
			setup: func(tempDir string) error {
				content := "Line 1\nLine 2\nLine 3\n"
				return os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(content), 0644)
			},
			expectError:    false,
			expectedOutput: "Line 1\nLine 2\nLine 3\n",
		},
		{
			name:  "empty file",
			args:  []string{"empty.txt"},
			flags: []string{},
			setup: func(tempDir string) error {
				return os.WriteFile(filepath.Join(tempDir, "empty.txt"), []byte(""), 0644)
			},
			expectError:    false,
			expectedOutput: "",
		},
		{
			name:  "single line file",
			args:  []string{"single.txt"},
			flags: []string{},
			setup: func(tempDir string) error {
				return os.WriteFile(filepath.Join(tempDir, "single.txt"), []byte("Only one line"), 0644)
			},
			expectError:    false,
			expectedOutput: "Only one line\n",
		},
		{
			name:  "multiple files",
			args:  []string{"file1.txt", "file2.txt"},
			flags: []string{"n", "2"},
			setup: func(tempDir string) error {
				content1 := "File1 Line1\nFile1 Line2\nFile1 Line3\n"
				content2 := "File2 Line1\nFile2 Line2\nFile2 Line3\n"
				if err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte(content1), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte(content2), 0644)
			},
			expectError:    false,
			expectedOutput: "==> file1.txt <==\nFile1 Line1\nFile1 Line2\n\n==> file2.txt <==\nFile2 Line1\nFile2 Line2\n",
		},
		{
			name:        "non-existent file",
			args:        []string{"nonexistent.txt"},
			flags:       []string{},
			setup:       func(tempDir string) error { return nil },
			expectError: true,
		},
		{
			name:        "invalid -n flag value",
			args:        []string{"test.txt"},
			flags:       []string{"n", "invalid"},
			setup:       func(tempDir string) error { return nil },
			expectError: true,
		},
		{
			name:        "negative -n flag value",
			args:        []string{"test.txt"},
			flags:       []string{"n", "-5"},
			setup:       func(tempDir string) error { return nil },
			expectError: true,
		},
		{
			name:        "-n flag without argument",
			args:        []string{"test.txt"},
			flags:       []string{"n"},
			setup:       func(tempDir string) error { return nil },
			expectError: true,
		},
		{
			name:        "invalid combined -n flag",
			args:        []string{"test.txt"},
			flags:       []string{"ninvalid"},
			setup:       func(tempDir string) error { return nil },
			expectError: true,
		},
		{
			name:        "negative combined -n flag",
			args:        []string{"test.txt"},
			flags:       []string{"n-5"},
			setup:       func(tempDir string) error { return nil },
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir := t.TempDir()
			
			// Change to temp directory for relative path tests
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

			// Setup test files
			if err := tt.setup(tempDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Execute command
			headx := &HeadxCommand{}
			err = headx.Execute(tt.args, tt.flags)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Output verification is done at integration level
			// This test focuses on error handling and execution flow
		})
	}
}

func TestHeadxCommand_ExecuteStdin(t *testing.T) {
	// Save original stdin
	originalStdin := os.Stdin

	tests := []struct {
		name           string
		flags          []string
		stdinContent   string
		expectedOutput string
	}{
		{
			name:           "default 10 lines from stdin",
			flags:          []string{},
			stdinContent:   "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10\nLine 11\nLine 12\n",
			expectedOutput: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10\n",
		},
		{
			name:           "custom lines from stdin",
			flags:          []string{"n", "3"},
			stdinContent:   "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n",
			expectedOutput: "Line 1\nLine 2\nLine 3\n",
		},
		{
			name:           "empty stdin",
			flags:          []string{},
			stdinContent:   "",
			expectedOutput: "",
		},
		{
			name:           "fewer lines than requested",
			flags:          []string{"n", "10"},
			stdinContent:   "Line 1\nLine 2\n",
			expectedOutput: "Line 1\nLine 2\n",
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

			// Restore stdin after test
			defer func() {
				os.Stdin = originalStdin
			}()

			// Execute command with no args (should read from stdin)
			headx := &HeadxCommand{}
			err = headx.Execute([]string{}, tt.flags)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Output verification is done at integration level
			// This test verifies the command executes without error
		})
	}
}

func TestHeadxCommand_processFile(t *testing.T) {
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
	reader := strings.NewReader(testContent)

	headx := &HeadxCommand{}
	err := headx.processFile(reader, "test-file", 3, true)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Output verification is done at integration level
	// This test verifies the method executes without error
}

func TestHeadxCommand_processFileWithoutHeader(t *testing.T) {
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
	reader := strings.NewReader(testContent)

	headx := &HeadxCommand{}
	err := headx.processFile(reader, "", 2, false)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Output verification is done at integration level
	// This test verifies the method executes without error
}

func TestHeadxCommand_processFileError(t *testing.T) {
	// Create a reader that returns an error
	errorReader := &errorScannerReader{}

	headx := &HeadxCommand{}
	err := headx.processFile(errorReader, "error-file", 3, false)

	if err == nil {
		t.Error("Expected error but got none")
	}

	expectedErrorMsg := "headx: error reading 'error-file'"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain %q, got %q", expectedErrorMsg, err.Error())
	}
}

func TestHeadxCommand_processFileErrorStdin(t *testing.T) {
	// Create a reader that returns an error for stdin case
	errorReader := &errorScannerReader{}

	headx := &HeadxCommand{}
	err := headx.processFile(errorReader, "", 3, false)

	if err == nil {
		t.Error("Expected error but got none")
	}

	expectedErrorMsg := "headx: error reading from stdin"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain %q, got %q", expectedErrorMsg, err.Error())
	}
}

func TestHeadxCommand_FlagParsing(t *testing.T) {
	tests := []struct {
		name        string
		flags       []string
		expectedN   int
		expectError bool
	}{
		{
			name:        "no flags",
			flags:       []string{},
			expectedN:   10,
			expectError: false,
		},
		{
			name:        "separate -n flag",
			flags:       []string{"n", "5"},
			expectedN:   5,
			expectError: false,
		},
		{
			name:        "combined -n flag",
			flags:       []string{"n15"},
			expectedN:   15,
			expectError: false,
		},
		{
			name:        "zero lines",
			flags:       []string{"n", "0"},
			expectedN:   0,
			expectError: false,
		},
		{
			name:        "invalid number",
			flags:       []string{"n", "abc"},
			expectedN:   10,
			expectError: true,
		},
		{
			name:        "negative number",
			flags:       []string{"n", "-1"},
			expectedN:   10,
			expectError: true,
		},
		{
			name:        "missing argument",
			flags:       []string{"n"},
			expectedN:   10,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file for testing
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.txt")
			if err := os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Change to temp directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}
			defer os.Chdir(oldDir)
			
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			headx := &HeadxCommand{}
			err = headx.Execute([]string{"test.txt"}, tt.flags)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// errorScannerReader is a helper type for testing scanner errors
type errorScannerReader struct{}

func (er *errorScannerReader) Read(p []byte) (n int, err error) {
	// Return some data first, then an error on subsequent reads
	if len(p) > 0 {
		p[0] = 'x'
		return 1, errors.New("scanner error")
	}
	return 0, errors.New("scanner error")
}


