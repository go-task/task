package fingerprint

import (
	"encoding/json"
	"fmt"

	"github.com/zeebo/xxh3"

	"github.com/go-task/task/v3/taskfile/ast"
)

type fingerprintIdentity struct {
	Task      string   `json:"task"`
	Dir       string   `json:"dir"`
	Sources   []string `json:"sources,omitempty"`
	Generates []string `json:"generates,omitempty"`
}

func taskFingerprintKey(t *ast.Task) string {
	name := taskIdentityName(t)
	identity := fingerprintIdentity{
		Task:      name,
		Dir:       t.Dir,
		Sources:   globPatterns(t.Sources),
		Generates: globPatterns(t.Generates),
	}

	encoded, err := json.Marshal(identity)
	if err != nil {
		return normalizeFilename(name)
	}

	return normalizeFilename(fmt.Sprintf("%s-%x", name, xxh3.Hash(encoded)))
}

func taskIdentityName(t *ast.Task) string {
	if t.FullName != "" {
		return t.FullName
	}
	return t.Task
}

func globPatterns(globs []*ast.Glob) []string {
	if len(globs) == 0 {
		return nil
	}

	patterns := make([]string, 0, len(globs))
	for _, glob := range globs {
		if glob == nil {
			continue
		}
		if glob.Negate {
			patterns = append(patterns, "!"+glob.Glob)
			continue
		}
		patterns = append(patterns, glob.Glob)
	}
	return patterns
}
