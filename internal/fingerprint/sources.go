package fingerprint

import "fmt"

func NewSourcesChecker(method, tempDir string, dry bool, rootDir string) (SourcesCheckable, error) {
	switch method {
	case "timestamp":
		return NewTimestampChecker(tempDir, dry, rootDir), nil
	case "checksum":
		return NewChecksumChecker(tempDir, dry, rootDir), nil
	case "none":
		return NoneChecker{}, nil
	default:
		return nil, fmt.Errorf(`task: invalid method "%s"`, method)
	}
}
