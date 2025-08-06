package fingerprint

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskfile/ast"
)

type TestDefinition struct {
	name     string
	setup    func(t *testing.T, dir string) *ast.Task
	expected bool
}

func TestTimestampCheckerIsUpToDate(t *testing.T) {
	t.Parallel()

	tests := []TestDefinition{
		{
			name: "empty sources",
			setup: func(t *testing.T, dir string) *ast.Task {
				return &ast.Task{
					Dir:       dir,
					Sources:   nil,
					Generates: nil,
				}
			},
			expected: false,
		},
		{
			name: "sources newer than generates",
			setup: func(t *testing.T, dir string) *ast.Task {
				// Create source file
				sourceFile := filepath.Join(dir, "source.txt")
				err := os.WriteFile(sourceFile, []byte("source"), 0644)
				require.NoError(t, err)

				// Create generate file with older timestamp
				generateFile := filepath.Join(dir, "generate.txt")
				err = os.WriteFile(generateFile, []byte("generate"), 0644)
				require.NoError(t, err)

				// Set source file to be newer than generate file
				sourceTime := time.Now()
				generateTime := sourceTime.Add(-1 * time.Hour)
				err = os.Chtimes(sourceFile, sourceTime, sourceTime)
				require.NoError(t, err)
				err = os.Chtimes(generateFile, generateTime, generateTime)
				require.NoError(t, err)

				return &ast.Task{
					Dir:       dir,
					Sources:   []*ast.Glob{{Glob: "source.txt"}},
					Generates: []*ast.Glob{{Glob: "generate.txt"}},
				}
			},
			expected: false,
		},
		{
			name: "generates newer than sources",
			setup: func(t *testing.T, dir string) *ast.Task {
				// Create source file
				sourceFile := filepath.Join(dir, "source.txt")
				err := os.WriteFile(sourceFile, []byte("source"), 0644)
				require.NoError(t, err)

				// Create generate file with newer timestamp
				generateFile := filepath.Join(dir, "generate.txt")
				err = os.WriteFile(generateFile, []byte("generate"), 0644)
				require.NoError(t, err)

				// Set generate file to be newer than source file
				sourceTime := time.Now().Add(-1 * time.Hour)
				generateTime := time.Now()
				err = os.Chtimes(sourceFile, sourceTime, sourceTime)
				require.NoError(t, err)
				err = os.Chtimes(generateFile, generateTime, generateTime)
				require.NoError(t, err)

				return &ast.Task{
					Dir:       dir,
					Sources:   []*ast.Glob{{Glob: "source.txt"}},
					Generates: []*ast.Glob{{Glob: "generate.txt"}},
				}
			},
			expected: true,
		},
		{
			name: "glob pattern directory/**/*",
			setup: func(t *testing.T, dir string) *ast.Task {
				// Create directory structure
				subDir := filepath.Join(dir, "subdir")
				nestedDir := filepath.Join(subDir, "nested")
				err := os.MkdirAll(nestedDir, 0755)
				require.NoError(t, err)

				// Create source files
				sourceFile1 := filepath.Join(subDir, "source1.txt")
				sourceFile2 := filepath.Join(nestedDir, "source2.txt")
				err = os.WriteFile(sourceFile1, []byte("source1"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(sourceFile2, []byte("source2"), 0644)
				require.NoError(t, err)

				// Create generate file
				generateFile := filepath.Join(dir, "generate.txt")
				generateFile2 := filepath.Join(dir, "generate2.txt")
				err = os.WriteFile(generateFile, []byte("generate"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(generateFile2, []byte("generate"), 0644)
				require.NoError(t, err)

				// Set source files to be newer than generate file to simulate a change
				generateTime := time.Now().Add(-1 * time.Hour)
				sourceTime := time.Now()
				err = os.Chtimes(sourceFile1, sourceTime, sourceTime)
				require.NoError(t, err)
				err = os.Chtimes(sourceFile2, sourceTime, sourceTime)
				require.NoError(t, err)
				err = os.Chtimes(generateFile, generateTime, generateTime)
				require.NoError(t, err)
				err = os.Remove(generateFile2) // Also remove one generate file to simulate a change
				require.NoError(t, err)

				return &ast.Task{
					Dir:       dir,
					Sources:   []*ast.Glob{{Glob: "subdir/**/*"}},
					Generates: []*ast.Glob{{Glob: "*.txt"}},
					Method:    "timestamp",
				}
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "task-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create test directory
			testDir := filepath.Join(tempDir, "test")
			err = os.MkdirAll(testDir, 0755)
			require.NoError(t, err)

			// Setup test
			task := tt.setup(t, testDir)

			// Create checker
			checker := NewTimestampChecker(tempDir, false)

			// Run test
			result, err := checker.IsUpToDate(task)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			// Verify timestamp file location if sources exist
			if len(task.Sources) > 0 {
				timestampFile := filepath.Join(tempDir, task.Dir, "timestamp", normalizeFilename(task.Task))
				if !tt.expected {
					// If task is not up-to-date, the timestamp file should still exist
					_, err := os.Stat(timestampFile)
					require.NoError(t, err, "Timestamp file should exist at %s", timestampFile)
				}
			}
		})
	}
}
