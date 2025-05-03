package taskrc

import (
	"os"

	"github.com/go-task/task/v3/taskrc/ast"
)

var defaultTaskRCs = []string{
	".taskrc.yml",
	".taskrc.yaml",
}

// GetConfig loads and merges local and global Task configuration files
func GetConfig(dir string) (*ast.TaskRC, error) {
	reader := NewReader()

	// LocalNode is the node for the local Task configuration file
	localNode, _ := NewNode("", dir)

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// GlobalNode is the node for the global Task configuration file (~/.taskrc.yml)
	globalNode, _ := NewNode("", home)

	localConfig, _ := reader.Read(localNode)

	globalConfig, _ := reader.Read(globalNode)

	if globalConfig == nil {
		return localConfig, nil
	}

	// Merge the global configuration into the local configuration
	globalConfig.Merge(localConfig)

	return globalConfig, nil
}
