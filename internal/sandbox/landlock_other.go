//go:build !linux

package sandbox

import "errors"

var ErrNotLinux error = errors.New("sandboxing only supported on Linux")

func WithSandbox(sources []string, generates []string) error {
	return ErrNotLinux
}
