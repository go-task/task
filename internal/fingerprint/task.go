package fingerprint

import (
	"context"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile"
)

type (
	CheckerOption func(*CheckerConfig)
	CheckerConfig struct {
		statusChecker  StatusCheckable
		sourcesChecker SourcesCheckable
	}
)

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
	t *taskfile.Task,
	method string,
	tempDir string,
	dry bool,
	logger *logger.Logger,
	opts ...CheckerOption,
) (bool, error) {
	var statusUpToDate bool
	var sourcesUpToDate bool
	var err error

	// If the task method is set, override the default method
	if t.Method != "" {
		method = t.Method
	}

	// Get the default checkers
	statusChecker := NewStatusChecker(logger)
	sourcesChecker, err := NewSourcesChecker(method, tempDir, dry)
	if err != nil {
		return false, err
	}

	// Default config
	config := &CheckerConfig{
		statusChecker:  statusChecker,
		sourcesChecker: sourcesChecker,
	}

	// Apply functional options
	for _, opt := range opts {
		opt(config)
	}

	statusIsSet := len(t.Status) != 0
	sourcesIsSet := len(t.Sources) != 0

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

	// If both status and sources are set, the task is up-to-date if both are up-to-date
	if statusIsSet && sourcesIsSet {
		return statusUpToDate && sourcesUpToDate, nil
	}

	// If only status is set, the task is up-to-date if the status is up-to-date
	if statusIsSet {
		return statusUpToDate, nil
	}

	// If only sources is set, the task is up-to-date if the sources are up-to-date
	if sourcesIsSet {
		return sourcesUpToDate, nil
	}

	// If no status or sources are set, the task should always run
	// i.e. it is never considered "up-to-date"
	return false, nil
}
