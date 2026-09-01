package main

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
)

func TestChangelogSection(t *testing.T) {
	t.Parallel()

	const changelog = `# Changelog

## v3.53.1 - 2026-08-18

### 🚀 Features

- Something new (#1 by @someone).

### 🐛 Fixes

- Something fixed (#2 by @someone).

## v3.53.10 - 2026-08-17

- An entry of a version whose heading starts with "## v3.53.1".

## v3.5.0 - 2026-01-01

- An older entry.
`

	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:    "section with sub-headings",
			version: "3.53.1",
			expected: `### 🚀 Features

- Something new (#1 by @someone).

### 🐛 Fixes

- Something fixed (#2 by @someone).`,
		},
		{
			name:     "last section of the file",
			version:  "3.5.0",
			expected: "- An older entry.",
		},
		{
			// A patch release may ship without changelog entries.
			name:     "missing section",
			version:  "3.53.2",
			expected: "",
		},
		{
			name:     "heading of another version starts with this one",
			version:  "3.53.10",
			expected: `- An entry of a version whose heading starts with "## v3.53.1".`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			version := semver.MustParse(test.version)
			assert.Equal(t, test.expected, changelogSection(changelog, version))
		})
	}
}
