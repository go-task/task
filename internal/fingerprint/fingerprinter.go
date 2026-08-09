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
	// resolution of the fingerprinting method and the checkers behind it.
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

// Kind names the fingerprint variable ("checksum", "timestamp" or "none") the
// resolved method injects. An invalid method is reported as "checksum" here and
// rejected by the entry points that build a checker.
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

// SourceValue returns the value of the fingerprint variable for the given task.
// It is potentially expensive, so only call it when the task references it.
func (f *Fingerprinter) SourceValue(t *ast.Task) (any, error) {
	sourcesChecker, err := f.resolveSourcesChecker(t)
	if err != nil {
		return nil, err
	}
	return sourcesChecker.Value(t)
}

// UpToDate reports whether the given task is up-to-date, considering both its
// status commands and its sources. A task that declares neither never is.
func (f *Fingerprinter) UpToDate(ctx context.Context, t *ast.Task) (bool, error) {
	var statusUpToDate bool
	var sourcesUpToDate bool

	statusChecker := f.statusChecker
	if statusChecker == nil {
		statusChecker = NewStatusChecker(f.logger)
	}
	sourcesChecker, err := f.resolveSourcesChecker(t)
	if err != nil {
		return false, err
	}

	statusIsSet := len(t.Status) != 0
	sourcesIsSet := len(t.Sources) != 0

	if statusIsSet {
		statusUpToDate, err = statusChecker.IsUpToDate(ctx, t)
		if err != nil {
			return false, err
		}
	}

	if sourcesIsSet {
		sourcesUpToDate, err = sourcesChecker.IsUpToDate(t)
		if err != nil {
			return false, err
		}
	}

	if statusIsSet && sourcesIsSet {
		return statusUpToDate && sourcesUpToDate, nil
	}
	if statusIsSet {
		return statusUpToDate, nil
	}
	if sourcesIsSet {
		return sourcesUpToDate, nil
	}
	return false, nil
}

// OnError gives the sources checker resolved for the given task a chance to
// clean up after a failed run.
func (f *Fingerprinter) OnError(t *ast.Task) error {
	sourcesChecker, err := f.resolveSourcesChecker(t)
	if err != nil {
		return err
	}
	return sourcesChecker.OnError(t)
}

// resolveSourcesChecker is the single place where a task is mapped to a checker.
func (f *Fingerprinter) resolveSourcesChecker(t *ast.Task) (SourcesCheckable, error) {
	if f.sourcesChecker != nil {
		return f.sourcesChecker, nil
	}
	return NewSourcesChecker(f.resolveMethod(t), f.tempDir, f.dry)
}
