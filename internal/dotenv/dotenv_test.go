package dotenv_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/internal/dotenv"
)

func TestRead(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"syntax":             "# comment\r\nexport Z=first\r\nA=${Z}/nested # comment\r\nM: yaml\r\nZ=last\r\nEMPTY=\r\n",
		"quotes":             "BASE=/home/user\nSINGLE='$BASE'\nDOUBLE=\"${BASE}/nested\\nnext\"\nMULTI='line one\nline two'\nESCAPED=\\$BASE\nTEMPLATE={{.BASE}}/nested",
		"invalid key":        "BAD-KEY=value\n",
		"unterminated quote": "VALUE=\"unfinished\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), ".env")
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
			expected, expectedErr := godotenv.Unmarshal(content)
			actual, err := dotenv.Read(path)
			if expectedErr != nil {
				require.EqualError(t, err, expectedErr.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, expected, maps.Collect(actual.AllFromFront()))
			if name == "syntax" {
				assert.Equal(t, []string{"Z", "A", "M", "EMPTY"}, slices.Collect(actual.Keys()))
			}
		})
	}
}

func TestReadMissingFile(t *testing.T) {
	t.Parallel()
	_, err := dotenv.Read(filepath.Join(t.TempDir(), "missing.env"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
