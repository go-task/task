package taskfile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadDotenvOrderedPreservesDeclarationOrder(t *testing.T) {
	t.Parallel()

	content := `# a comment
export BASE_DIR=/home/user
MID_DIR={{.BASE_DIR}}/nested
FINAL_DIR="{{.MID_DIR}}/deeper"
FULL_PATH=$FINAL_DIR/file.txt

QUOTED='single quoted'
FULL_PATH=$FINAL_DIR/overridden.txt
`

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	// Run several times to make sure the order is always the same. This
	// mirrors the previous godotenv.Read-based implementation, which built
	// its result in a plain Go map and iterated over it in random order.
	for range 20 {
		envs, err := ReadDotenvOrdered(path)
		require.NoError(t, err)

		assert.Equal(t,
			[]string{"BASE_DIR", "MID_DIR", "FINAL_DIR", "FULL_PATH", "QUOTED"},
			slices.Collect(envs.Keys()),
		)

		baseDir, _ := envs.Get("BASE_DIR")
		assert.Equal(t, "/home/user", baseDir)

		midDir, _ := envs.Get("MID_DIR")
		assert.Equal(t, "{{.BASE_DIR}}/nested", midDir)

		finalDir, _ := envs.Get("FINAL_DIR")
		assert.Equal(t, "{{.MID_DIR}}/deeper", finalDir)

		// Last declaration of a duplicate key wins, but keeps its original
		// position in the order. $FINAL_DIR is expanded eagerly, using
		// FINAL_DIR's raw (not yet Go-template-expanded) value.
		fullPath, _ := envs.Get("FULL_PATH")
		assert.Equal(t, "{{.MID_DIR}}/deeper/overridden.txt", fullPath)

		quoted, _ := envs.Get("QUOTED")
		assert.Equal(t, "single quoted", quoted)
	}
}
