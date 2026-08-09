package fingerprint

import (
	"fmt"

	"github.com/go-task/task/v3/errors"
)

// ErrInvalidMethod marks a method name that maps to no checker. Callers that
// only need a fingerprint value can tell it apart from a checker failing on
// the sources themselves.
var ErrInvalidMethod = errors.New("invalid method")

func NewSourcesChecker(method, tempDir string, dry bool) (SourcesCheckable, error) {
	switch method {
	case "timestamp":
		return NewTimestampChecker(tempDir, dry), nil
	case "checksum":
		return NewChecksumChecker(tempDir, dry), nil
	case "none":
		return NoneChecker{}, nil
	default:
		return nil, fmt.Errorf(`task: %w "%s"`, ErrInvalidMethod, method)
	}
}
