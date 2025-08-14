package taskrc

import (
	"os"
	"slices"

	"github.com/go-task/task/v3/internal/fsext"
	"github.com/go-task/task/v3/taskrc/ast"
)

var defaultTaskRCs = []string{
	".taskrc.yml",
	".taskrc.yaml",
}

// GetConfig loads and merges local and global Task configuration files
func GetConfig(dir string) (*ast.TaskRC, error) {
	var config *ast.TaskRC
	reader := NewReader()

	// Read the XDG config file
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		xdgConfigNode, err := NewNode("", xdgConfigHome)
		if err == nil && xdgConfigNode != nil {
			config, err = reader.Read(xdgConfigNode)
			if err != nil {
				return nil, err
			}
		}
	}

	// Find all the nodes from the given directory up to the users home directory
	entrypoints, err := fsext.SearchAll("", dir, defaultTaskRCs)
	if err != nil {
		return nil, err
	}

	// Reverse the entrypoints since we want the child files to override parent ones
	slices.Reverse(entrypoints)

	// Loop over the nodes, and merge them into the main config
	for _, entrypoint := range entrypoints {
		node, err := NewNode("", entrypoint)
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
