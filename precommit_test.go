package task_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

type preCommitHook struct {
	ID                        string   `yaml:"id"`
	Name                      string   `yaml:"name"`
	Description               string   `yaml:"description"`
	Language                  string   `yaml:"language"`
	Entry                     string   `yaml:"entry"`
	PassFilenames             *bool    `yaml:"pass_filenames"`
	RequireSerial             *bool    `yaml:"require_serial"`
	MinimumPreCommitVersion   string   `yaml:"minimum_pre_commit_version"`
	Types                     []string `yaml:"types"`
	Args                      []string `yaml:"args"`
}

func TestPreCommitHooksFile(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(".pre-commit-hooks.yaml")
	require.NoError(t, err, ".pre-commit-hooks.yaml should exist")

	var hooks []preCommitHook
	err = yaml.Unmarshal(data, &hooks)
	require.NoError(t, err, ".pre-commit-hooks.yaml should be valid YAML")

	require.Len(t, hooks, 1, "should define exactly one hook")

	hook := hooks[0]
	assert.Equal(t, "task", hook.ID, "hook id should be 'task'")
	assert.NotEmpty(t, hook.Name, "hook name should not be empty")
	assert.NotEmpty(t, hook.Description, "hook description should not be empty")
	assert.Equal(t, "golang", hook.Language, "hook language should be 'golang'")
	assert.Equal(t, "task", hook.Entry, "hook entry should be 'task'")
	require.NotNil(t, hook.PassFilenames, "pass_filenames should be set")
	assert.False(t, *hook.PassFilenames, "pass_filenames should be false")
	require.NotNil(t, hook.RequireSerial, "require_serial should be set")
	assert.True(t, *hook.RequireSerial, "require_serial should be true")
	assert.Equal(t, "3.0.0", hook.MinimumPreCommitVersion, "minimum_pre_commit_version should be '3.0.0'")
}
