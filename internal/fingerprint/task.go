package fingerprint

import (
	"context"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

type (
	CheckerOption func(*CheckerConfig)
	CheckerConfig struct {
		method            string
		dry               bool
		tempDir           string
		logger            *logger.Logger
		definitionChecker DefinitionCheckable
		statusChecker     StatusCheckable
		sourcesChecker    SourcesCheckable
	}
)

func WithMethod(method string) CheckerOption {
	return func(config *CheckerConfig) {
		config.method = method
	}
}

func WithDry(dry bool) CheckerOption {
	return func(config *CheckerConfig) {
		config.dry = dry
	}
}

func WithTempDir(tempDir string) CheckerOption {
	return func(config *CheckerConfig) {
		config.tempDir = tempDir
	}
}

func WithLogger(logger *logger.Logger) CheckerOption {
	return func(config *CheckerConfig) {
		config.logger = logger
	}
}

func WithStatusChecker(checker StatusCheckable) CheckerOption {
	return func(config *CheckerConfig) {
		config.statusChecker = checker
	}
}

func WithSourcesChecker(checker SourcesCheckable) CheckerOption {
	return func(config *CheckerConfig) {
		config.sourcesChecker = checker
	}
}

func IsTaskUpToDate(
	ctx context.Context,
	t *ast.Task,
	opts ...CheckerOption,
) (bool, error) {
	var statusUpToDate bool
	var sourcesUpToDate bool
	var err error

	// Default config
	config := &CheckerConfig{
		method:            "none",
		tempDir:           "",
		dry:               false,
		logger:            nil,
		definitionChecker: nil,
		statusChecker:     nil,
		sourcesChecker:    nil,
	}

	// Apply functional options
	for _, opt := range opts {
		opt(config)
	}

	// If no status checker was given, set up the default one
	if config.statusChecker == nil {
		config.statusChecker = NewStatusChecker(config.logger)
	}

	// If no sources checker was given, set up the default one
	if config.sourcesChecker == nil {
		config.sourcesChecker, err = NewSourcesChecker(config.method, config.tempDir, config.dry)
		if err != nil {
			return false, err
		}
	}

	// if no definition checker was given, set up the default one
	if config.definitionChecker == nil {
		config.definitionChecker = NewDefinitionChecker(config.tempDir, config.dry, config.logger)
	}

	statusIsSet := len(t.Status) != 0
	sourcesIsSet := len(t.Sources) != 0

	// hash the task definition
	maybeDefinitionPath, err := config.definitionChecker.HashDefinition(t)
	if err != nil {
		return false, err
	}

	// if the status or sources are set, check if the definition is up-to-date
	// TODO: allow caching based on the task definition even if status or sources are not set
	if sourcesIsSet || statusIsSet {
		// check if the definition is up-to-date
		isDefinitionUpToDate, err := config.definitionChecker.IsUpToDate(maybeDefinitionPath)
		if err != nil {
			return false, err
		}
		// defintion is not up-to-date, early return
		if !isDefinitionUpToDate {
			return false, nil
		}
	}

	// If status is set, check if it is up-to-date
	if statusIsSet {
		statusUpToDate, err = config.statusChecker.IsUpToDate(ctx, t)
		if err != nil {
			return false, err
		}
	}

	// If sources is set, check if they are up-to-date
	if sourcesIsSet {
		sourcesUpToDate, err = config.sourcesChecker.IsUpToDate(t)
		if err != nil {
			return false, err
		}
	}

	// If no status or sources are set, the task should always run
	// i.e. it is never considered "up-to-date"
	isUpToDate := false

	// If both status and sources are set, the task is up-to-date if both are up-to-date
	if statusIsSet && sourcesIsSet {
		isUpToDate = statusUpToDate && sourcesUpToDate
	} else if statusIsSet {
		// If only status is set, the task is up-to-date if the status is up-to-date
		isUpToDate = statusUpToDate
	} else if sourcesIsSet {
		// If only sources is set, the task is up-to-date if the sources are up-to-date
		isUpToDate = sourcesUpToDate
	}

	if !isUpToDate {
		// if the task is not up-to-date for any reason, remove the definition file from previous runs if it exists
		err = config.definitionChecker.Cleanup(maybeDefinitionPath)
		if err != nil {
			return false, err
		}
	}

	return isUpToDate, nil
}
