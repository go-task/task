package taskrc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestGetConfig_RemoteTrustedHosts(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	_, _, localDir := setupDirs(t)

	// Test with single host
	configYAML := `
remote:
  trusted-hosts:
    - github.com
`
	writeFile(t, localDir, ".taskrc.yml", configYAML)

	cfg, err := GetConfig(localDir)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, []string{"github.com"}, cfg.Remote.TrustedHosts)

	// Test with multiple hosts
	configYAML = `
remote:
  trusted-hosts:
    - github.com
    - gitlab.com
    - example.com:8080
`
	writeFile(t, localDir, ".taskrc.yml", configYAML)

	cfg, err = GetConfig(localDir)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, []string{"github.com", "gitlab.com", "example.com:8080"}, cfg.Remote.TrustedHosts)
}

func TestGetConfig_RemoteTrustedHostsMerge(t *testing.T) { //nolint:paralleltest // cannot run in parallel
	t.Run("file-based merge precedence", func(t *testing.T) { //nolint:paralleltest // parent test cannot run in parallel
		xdgConfigDir, homeDir, localDir := setupDirs(t)

		// XDG config has github.com and gitlab.com
		xdgConfig := `
remote:
  trusted-hosts:
    - github.com
    - gitlab.com
  timeout: "30s"
`
		writeFile(t, xdgConfigDir, "taskrc.yml", xdgConfig)

		// Home config has example.com (should be combined with XDG)
		homeConfig := `
remote:
  trusted-hosts:
    - example.com
`
		writeFile(t, homeDir, ".taskrc.yml", homeConfig)

		cfg, err := GetConfig(localDir)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		// Home config entries come first, then XDG
		assert.Equal(t, []string{"example.com", "github.com", "gitlab.com"}, cfg.Remote.TrustedHosts)

		// Test with local config too
		localConfig := `
remote:
  trusted-hosts:
    - local.dev
`
		writeFile(t, localDir, ".taskrc.yml", localConfig)

		cfg, err = GetConfig(localDir)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		// Local config entries come first
		assert.Equal(t, []string{"example.com", "github.com", "gitlab.com", "local.dev"}, cfg.Remote.TrustedHosts)
	})

	t.Run("merge edge cases", func(t *testing.T) { //nolint:paralleltest // parent test cannot run in parallel
		tests := []struct {
			name     string
			base     *ast.TaskRC
			other    *ast.TaskRC
			expected []string
		}{
			{
				name: "merge hosts into empty",
				base: &ast.TaskRC{},
				other: &ast.TaskRC{
					Remote: ast.Remote{
						TrustedHosts: []string{"github.com"},
					},
				},
				expected: []string{"github.com"},
			},
			{
				name: "merge combines lists",
				base: &ast.TaskRC{
					Remote: ast.Remote{
						TrustedHosts: []string{"base.com"},
					},
				},
				other: &ast.TaskRC{
					Remote: ast.Remote{
						TrustedHosts: []string{"other.com"},
					},
				},
				expected: []string{"base.com", "other.com"},
			},
			{
				name: "merge empty list does not override",
				base: &ast.TaskRC{
					Remote: ast.Remote{
						TrustedHosts: []string{"base.com"},
					},
				},
				other: &ast.TaskRC{
					Remote: ast.Remote{
						TrustedHosts: []string{},
					},
				},
				expected: []string{"base.com"},
			},
			{
				name: "merge nil does not override",
				base: &ast.TaskRC{
					Remote: ast.Remote{
						TrustedHosts: []string{"base.com"},
					},
				},
				other: &ast.TaskRC{
					Remote: ast.Remote{
						TrustedHosts: nil,
					},
				},
				expected: []string{"base.com"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // parent test cannot run in parallel
				tt.base.Merge(tt.other)
				assert.Equal(t, tt.expected, tt.base.Remote.TrustedHosts)
			})
		}
	})

	t.Run("all remote fields merge", func(t *testing.T) { //nolint:paralleltest // parent test cannot run in parallel
		insecureTrue := true
		offlineTrue := true
		timeout := 30 * time.Second
		cacheExpiry := 1 * time.Hour

		base := &ast.TaskRC{}
		other := &ast.TaskRC{
			Remote: ast.Remote{
				Insecure:     &insecureTrue,
				Offline:      &offlineTrue,
				Timeout:      &timeout,
				CacheExpiry:  &cacheExpiry,
				TrustedHosts: []string{"github.com", "gitlab.com"},
			},
		}

		base.Merge(other)

		assert.Equal(t, &insecureTrue, base.Remote.Insecure)
		assert.Equal(t, &offlineTrue, base.Remote.Offline)
		assert.Equal(t, &timeout, base.Remote.Timeout)
		assert.Equal(t, &cacheExpiry, base.Remote.CacheExpiry)
		assert.Equal(t, []string{"github.com", "gitlab.com"}, base.Remote.TrustedHosts)
	})
}
