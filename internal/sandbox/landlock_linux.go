//go:build linux

package sandbox

import (
	"os"

	"github.com/landlock-lsm/go-landlock/landlock"
)

func WithSandbox(sources []string, generates []string) error {
	defaultAllowRW := []string{
		".task",
		"/tmp",
	}

	var allowList []landlock.PathOpt
	allowList = append(allowList, landlock.RODirs(defaultAllowRW...))
	allowList = append(allowList, parseSources(sources)...)
	allowList = append(allowList, parseGenerates(generates)...)

	return landlock.V2.BestEffort().RestrictPaths(allowList...)
}

func parseSources(sources []string) []landlock.PathOpt {
	var opts []landlock.PathOpt

	for _, source := range sources {
		if stat, err := os.Stat(source); err == nil {
			if stat.IsDir() {
				opts = append(opts, landlock.RODirs(source))
			}

			opts = append(opts, landlock.ROFiles(source))
		}
	}

	return opts
}

func parseGenerates(generates []string) []landlock.PathOpt {
	var opts []landlock.PathOpt

	for _, generate := range generates {
		if stat, err := os.Stat(generate); err == nil {
			if stat.IsDir() {
				opts = append(opts, landlock.RWDirs(generate))
			}

			opts = append(opts, landlock.RWFiles(generate))
		}
	}

	return opts
}
