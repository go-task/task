package taskrc

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-task/task/v3/taskrc/ast"
)

const (
	localConfigYAML = `
experiments:
  GENTLE_FORCE: 1
  ENV_PRECEDENCE: 0
`

	globalConfigYAML = `
experiments:
  GENTLE_FORCE: 0
  REMOTE_TASKFILES: 1
  ENV_PRECEDENCE: 1

`
)

func setupDirs(t *testing.T) (localDir, globalDir string) {
	t.Helper()
	localDir = t.TempDir()
	globalDir = t.TempDir()

	t.Setenv("HOME", globalDir)

	return localDir, globalDir
}

func writeFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644)
	assert.NoError(t, err)
}

func TestGetConfig_MergesGlobalAndLocal(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	localDir, globalDir := setupDirs(t)

	// Write local config
	writeFile(t, localDir, ".taskrc.yml", localConfigYAML)

	// Write global config
	writeFile(t, globalDir, ".taskrc.yml", globalConfigYAML)

	cfg, err := GetConfig(localDir)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	fmt.Printf("cfg : %#v\n", cfg)
	assert.Equal(t, &ast.TaskRC{Version: nil, Experiments: map[string]int{"GENTLE_FORCE": 1, "ENV_PRECEDENCE": 0, "REMOTE_TASKFILES": 1}}, cfg)
}

func TestGetConfig_NoConfigFiles(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	localDir, _ := setupDirs(t)

	cfg, err := GetConfig(localDir)
	fmt.Printf("cfg : %#v\n", cfg)
	assert.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestGetConfig_OnlyGlobal(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	localDir, globalDir := setupDirs(t)

	writeFile(t, globalDir, ".taskrc.yml", globalConfigYAML)

	cfg, err := GetConfig(localDir)
	assert.NoError(t, err)
	assert.Equal(t, &ast.TaskRC{Version: nil, Experiments: map[string]int{"GENTLE_FORCE": 0, "ENV_PRECEDENCE": 1, "REMOTE_TASKFILES": 1}}, cfg)
}

func TestGetConfig_OnlyLocal(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	localDir, _ := setupDirs(t)

	writeFile(t, localDir, ".taskrc.yml", localConfigYAML)

	cfg, err := GetConfig(localDir)
	assert.NoError(t, err)
	assert.Equal(t, &ast.TaskRC{Version: nil, Experiments: map[string]int{"GENTLE_FORCE": 1, "ENV_PRECEDENCE": 0}}, cfg)
}
