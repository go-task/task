package taskrc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/taskrc/ast"
)

const (
	xdgConfigYAML = `
experiments:
  FOO: 1
  BAR: 1
  BAZ: 1
`

	homeConfigYAML = `
experiments:
  FOO: 2
  BAR: 2
`

	localConfigYAML = `
experiments:
  FOO: 3
`
)

func setupDirs(t *testing.T) (string, string, string) {
	t.Helper()

	xdgConfigDir := t.TempDir()
	xdgTaskConfigDir := filepath.Join(xdgConfigDir, "task")
	require.NoError(t, os.Mkdir(xdgTaskConfigDir, 0o755))

	homeDir := t.TempDir()

	localDir := filepath.Join(homeDir, "local")
	require.NoError(t, os.Mkdir(localDir, 0o755))

	t.Setenv("XDG_CONFIG_HOME", xdgConfigDir)
	t.Setenv("HOME", homeDir)

	return xdgTaskConfigDir, homeDir, localDir
}

func writeFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644)
	assert.NoError(t, err)
}

func TestGetConfig_NoConfigFiles(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	_, _, localDir := setupDirs(t)

	cfg, err := GetConfig(localDir)
	assert.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestGetConfig_OnlyXDG(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	xdgDir, _, localDir := setupDirs(t)

	writeFile(t, xdgDir, "taskrc.yml", xdgConfigYAML)

	cfg, err := GetConfig(localDir)
	assert.NoError(t, err)
	assert.Equal(t, &ast.TaskRC{
		Version: nil,
		Experiments: map[string]int{
			"FOO": 1,
			"BAR": 1,
			"BAZ": 1,
		},
	}, cfg)
}

func TestGetConfig_OnlyHome(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	_, homeDir, localDir := setupDirs(t)

	writeFile(t, homeDir, ".taskrc.yml", homeConfigYAML)

	cfg, err := GetConfig(localDir)
	assert.NoError(t, err)
	assert.Equal(t, &ast.TaskRC{
		Version: nil,
		Experiments: map[string]int{
			"FOO": 2,
			"BAR": 2,
		},
	}, cfg)
}

func TestGetConfig_OnlyLocal(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	_, _, localDir := setupDirs(t)

	writeFile(t, localDir, ".taskrc.yml", localConfigYAML)

	cfg, err := GetConfig(localDir)
	assert.NoError(t, err)
	assert.Equal(t, &ast.TaskRC{
		Version: nil,
		Experiments: map[string]int{
			"FOO": 3,
		},
	}, cfg)
}

func TestGetConfig_All(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	xdgConfigDir, homeDir, localDir := setupDirs(t)

	// Write local config
	writeFile(t, localDir, ".taskrc.yml", localConfigYAML)

	// Write home config
	writeFile(t, homeDir, ".taskrc.yml", homeConfigYAML)

	// Write XDG config
	writeFile(t, xdgConfigDir, "taskrc.yml", xdgConfigYAML)

	cfg, err := GetConfig(localDir)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, &ast.TaskRC{
		Version: nil,
		Experiments: map[string]int{
			"FOO": 3,
			"BAR": 2,
			"BAZ": 1,
		},
	}, cfg)
}
