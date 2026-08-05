package fingerprint

import (
	"context"

	"github.com/go-task/task/v3/internal/logger"
	"github.com/go-task/task/v3/taskfile/ast"
)

type (
	// A FingerprinterOption is a functional option for a [Fingerprinter].
	FingerprinterOption func(*Fingerprinter)

	// A Fingerprinter answers whether a task is up-to-date. It owns the
	// resolution of the fingerprinting method (the task's method, falling back
	// to the default) and the construction of the underlying checkers, so that
	// every caller gets the same answer for the same task.
	Fingerprinter struct {
		defaultMethod  string
		tempDir        string
		dry            bool
		logger         *logger.Logger
		statusChecker  StatusCheckable
		sourcesChecker SourcesCheckable
	}
)

// WithStatusChecker allows a custom [StatusCheckable] to be used instead of
// the default one.
func WithStatusChecker(checker StatusCheckable) FingerprinterOption {
	return func(f *Fingerprinter) {
		f.statusChecker = checker
	}
}

// WithSourcesChecker allows a custom [SourcesCheckable] to be used instead of
// the one selected by the resolved fingerprinting method.
func WithSourcesChecker(checker SourcesCheckable) FingerprinterOption {
	return func(f *Fingerprinter) {
		f.sourcesChecker = checker
	}
}

// NewFingerprinter creates a new [Fingerprinter]. The defaultMethod is used
// for tasks that don't declare a method of their own.
func NewFingerprinter(
	defaultMethod string,
	tempDir string,
	dry bool,
	logger *logger.Logger,
	opts ...FingerprinterOption,
) *Fingerprinter {
	f := &Fingerprinter{
		defaultMethod: defaultMethod,
		tempDir:       tempDir,
		dry:           dry,
		logger:        logger,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *Fingerprinter) resolveMethod(t *ast.Task) string {
	if t.Method != "" {
		return t.Method
	}
	return f.defaultMethod
}

// Kind returns the kind of fingerprint variable ("checksum", "timestamp" or
// "none") produced by the method resolved for the given task. Unknown methods
// fall back to "checksum"; they are only rejected by [NewSourcesChecker].
func (f *Fingerprinter) Kind(t *ast.Task) string {
	if f.sourcesChecker != nil {
		return f.sourcesChecker.Kind()
	}
	switch method := f.resolveMethod(t); method {
	case "timestamp", "none":
		return method
	default:
		return "checksum"
	}
}

// SourceValue returns the value of the fingerprint variable (CHECKSUM or
// TIMESTAMP) for the given task. It is potentially expensive, so callers
// should only invoke it when the task actually references the variable.
func (f *Fingerprinter) SourceValue(t *ast.Task) (any, error) {
	sourcesChecker, err := f.resolveSourcesChecker(f.Kind(t))
	if err != nil {
		return nil, err
	}
	return sourcesChecker.Value(t)
}

// UpToDate reports whether the given task is up-to-date, considering both its
// status commands and its sources.
//
// | Status up-to-date | Sources up-to-date | Task is up-to-date |
// | ----------------- | ------------------ | ------------------ |
// | not set           | not set            | false              |
// | not set           | true               | true               |
// | not set           | false              | false              |
// | true              | not set            | true               |
// | true              | true               | true               |
// | true              | false              | false              |
// | false             | not set            | false              |
// | false             | true               | false              |
// | false             | false              | false              |
func (f *Fingerprinter) UpToDate(ctx context.Context, t *ast.Task) (bool, error) {
	var statusUpToDate bool
	var sourcesUpToDate bool

	statusChecker := f.statusChecker
	if statusChecker == nil {
		statusChecker = NewStatusChecker(f.logger)
	}
	sourcesChecker, err := f.resolveSourcesChecker(f.resolveMethod(t))
	if err != nil {
		return false, err
	}

	statusIsSet := len(t.Status) != 0
	sourcesIsSet := len(t.Sources) != 0

	// If status is set, check if it is up-to-date
	if statusIsSet {
		statusUpToDate, err = statusChecker.IsUpToDate(ctx, t)
		if err != nil {
			return false, err
		}
	}

	// If sources is set, check if they are up-to-date
	if sourcesIsSet {
		sourcesUpToDate, err = sourcesChecker.IsUpToDate(t)
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

// OnError gives the sources checker resolved for the given task a chance to
// clean up after a failed run.
func (f *Fingerprinter) OnError(t *ast.Task) error {
	sourcesChecker, err := f.resolveSourcesChecker(f.resolveMethod(t))
	if err != nil {
		return err
	}
	return sourcesChecker.OnError(t)
}

func (f *Fingerprinter) resolveSourcesChecker(method string) (SourcesCheckable, error) {
	if f.sourcesChecker != nil {
		return f.sourcesChecker, nil
	}
	return NewSourcesChecker(method, f.tempDir, f.dry)
}
