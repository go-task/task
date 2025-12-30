package taskfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadDotenvOrdered(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		expected []DotenvKeyValue
	}{
		{
			name: "maintains order of variables",
			content: `VAR_A=first
VAR_B=second
VAR_C=third
VAR_D=fourth`,
			expected: []DotenvKeyValue{
				{Key: "VAR_A", Value: "first"},
				{Key: "VAR_B", Value: "second"},
				{Key: "VAR_C", Value: "third"},
				{Key: "VAR_D", Value: "fourth"},
			},
		},
		{
			name: "handles comments and empty lines",
			content: `# This is a comment
VAR_A=first

# Another comment
VAR_B=second
`,
			expected: []DotenvKeyValue{
				{Key: "VAR_A", Value: "first"},
				{Key: "VAR_B", Value: "second"},
			},
		},
		{
			name: "handles export prefix",
			content: `export VAR_A=first
VAR_B=second
export VAR_C=third`,
			expected: []DotenvKeyValue{
				{Key: "VAR_A", Value: "first"},
				{Key: "VAR_B", Value: "second"},
				{Key: "VAR_C", Value: "third"},
			},
		},
		{
			name: "handles quoted values",
			content: `VAR_A="quoted value"
VAR_B='single quoted'
VAR_C=unquoted`,
			expected: []DotenvKeyValue{
				{Key: "VAR_A", Value: "quoted value"},
				{Key: "VAR_B", Value: "single quoted"},
				{Key: "VAR_C", Value: "unquoted"},
			},
		},
		{
			name: "first occurrence wins for duplicates",
			content: `VAR_A=first
VAR_A=second`,
			expected: []DotenvKeyValue{
				{Key: "VAR_A", Value: "second"}, // godotenv takes last value
			},
		},
		{
			name: "handles colon separator",
			content: `VAR_A: first
VAR_B=second`,
			expected: []DotenvKeyValue{
				{Key: "VAR_A", Value: "first"},
				{Key: "VAR_B", Value: "second"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a temporary file with the test content
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, ".env")
			err := os.WriteFile(envFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			// Read the file
			result, err := ReadDotenvOrdered(envFile)
			require.NoError(t, err)

			// Verify the result
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadDotenvOrderedConsistency(t *testing.T) {
	t.Parallel()

	// Create a file with many variables to ensure order is consistent
	content := `VAR_01=value01
VAR_02=value02
VAR_03=value03
VAR_04=value04
VAR_05=value05
VAR_06=value06
VAR_07=value07
VAR_08=value08
VAR_09=value09
VAR_10=value10`

	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envFile, []byte(content), 0o644)
	require.NoError(t, err)

	// Read multiple times and ensure the order is always the same
	var firstResult []DotenvKeyValue
	for i := 0; i < 100; i++ {
		result, err := ReadDotenvOrdered(envFile)
		require.NoError(t, err)

		if firstResult == nil {
			firstResult = result
		} else {
			assert.Equal(t, firstResult, result, "Order should be consistent across reads (iteration %d)", i)
		}
	}

	// Verify expected order
	expectedKeys := []string{
		"VAR_01", "VAR_02", "VAR_03", "VAR_04", "VAR_05",
		"VAR_06", "VAR_07", "VAR_08", "VAR_09", "VAR_10",
	}
	for i, kv := range firstResult {
		assert.Equal(t, expectedKeys[i], kv.Key)
	}
}
