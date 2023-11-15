package fingerprint

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
	"github.com/mitchellh/hashstructure/v2"
)

// DefinitionChecker checks if the task definition and any of its variables/environment variables change.
type DefinitionChecker struct {
	tempDir string
	dry     bool
	logger  *logger.Logger
}

func NewDefinitionChecker(tempDir string, dry bool, logger *logger.Logger) *DefinitionChecker {
	return &DefinitionChecker{
		tempDir: tempDir,
		dry:     dry,
		logger:  logger,
	}
}

// IsUpToDate returns true if the task definition is up-to-date.
// As the second return value, it returns the path to the task definition file if it exists.
// This file should be cleaned up if the task is not up-to-date by other checkers.
func (checker *DefinitionChecker) IsUpToDate(maybeDefinitionPath *string) (bool, error) {
	if maybeDefinitionPath == nil {
		return false, fmt.Errorf("task: task definition path is nil")
	}
	definitionPath := *maybeDefinitionPath

	// check if the file exists
	_, err := os.Stat(definitionPath)
	if err == nil {
		checker.logger.VerboseOutf(logger.Magenta, "task: task definition is up-to-date: %s\n", definitionPath)
		// file exists, the task definition is up to
		return true, nil
	}

	// task is not up-to-date as the file does not exist
	// create the hash file if not in dry mode
	if !checker.dry {
		// create the file
		if err := os.MkdirAll(filepath.Dir(definitionPath), 0o755); err != nil {
			return false, err
		}
		_, err = os.Create(definitionPath)
		if err != nil {
			return false, err
		}
		checker.logger.VerboseOutf(logger.Yellow, "task: task definition was written as: %s\n", definitionPath)
	}
	return false, nil
}

func (checker *DefinitionChecker) HashDefinition(t *taskfile.Task) (*string, error) {
	// hash the task
	hash, err := hashstructure.Hash(t, hashstructure.FormatV2, nil)
	if err != nil {
		// failed to hash the task. Consider the task as not up-to-date
		return nil, err
	}

	// the path to the task definition file with the hash in the filename
	hashPath := filepath.Join(checker.tempDir, "definition", normalizeFilename(fmt.Sprintf("%s-%d", t.Name(), hash)))
	return &hashPath, nil
}

// Cleanup removes the task definition file if it exists.
func (checker *DefinitionChecker) Cleanup(definitionPath *string) error {
	if definitionPath != nil {
		// if the file exists, remove it
		if _, err := os.Stat(*definitionPath); err == nil {
			if err := os.Remove(*definitionPath); err != nil {
				return err
			}
		}
	}
	return nil
}
