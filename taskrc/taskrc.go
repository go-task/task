package taskrc

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-task/task/v3/internal/fsext"
	"github.com/go-task/task/v3/taskrc/ast"
)

var (
	defaultXDGTaskRCs = []string{
		"taskrc.yml",
		"taskrc.yaml",
	}
	defaultTaskRCs = []string{
		".taskrc.yml",
		".taskrc.yaml",
	}
)

// GetConfig loads and merges local and global Task configuration files
func GetConfig(dir string) (*ast.TaskRC, error) {
	var config *ast.TaskRC
	reader := NewReader()

	// Read the XDG config file
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		xdgConfigNode, err := NewNode("", filepath.Join(xdgConfigHome, "task"), defaultXDGTaskRCs)
		if err == nil && xdgConfigNode != nil {
			xdgConfig, err := reader.Read(xdgConfigNode)
			if err != nil {
				return nil, err
			}
			config = xdgConfig
		}
	}

	// If the current path does not contain $HOME
	// If it does contain $HOME, then we will find this config later anyway
	home, err := os.UserHomeDir()
	if err == nil && !strings.Contains(home, dir) {
		homeNode, err := NewNode("", home, defaultTaskRCs)
		if err == nil && homeNode != nil {
			homeConfig, err := reader.Read(homeNode)
			if err != nil {
				return nil, err
			}
			if config == nil {
				config = homeConfig
			} else {
				config.Merge(homeConfig)
			}
		}
	}

	// Find all the nodes from the given directory up to the users home directory
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return config, err
	}
	entrypoints, err := fsext.SearchAll("", absDir, defaultTaskRCs)
	if err != nil {
		return config, err
	}

	// Reverse the entrypoints since we want the child files to override parent ones
	slices.Reverse(entrypoints)

	// Loop over the nodes, and merge them into the main config
	for _, entrypoint := range entrypoints {
		node, err := NewNode("", entrypoint, defaultTaskRCs)
		if err != nil {
			return nil, err
		}
		localConfig, err := reader.Read(node)
		if err != nil {
			return nil, err
		}
		if localConfig == nil {
			continue
		}
		if config == nil {
			config = localConfig
			continue
		}
		config.Merge(localConfig)
	}
	return config, nil
}
