package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// notes writes the changelog entries of the version being released, for the
// release pipeline to use as the body of its GitHub release.
func notes(w io.Writer) error {
	version, err := getVersion(versionFile)
	if err != nil {
		return err
	}

	b, err := os.ReadFile(changelogSource)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, changelogSection(string(b), version))
	return err
}

// changelogSection returns the body of the "## v<version> - <date>" section,
// heading excluded: the GitHub release already displays the version and date.
// The section is empty for a release without changelog entries.
func changelogSection(changelog string, version *semver.Version) string {
	heading := fmt.Sprintf("## v%s ", version)
	var section []string
	var found bool
	for line := range strings.SplitSeq(changelog, "\n") {
		if strings.HasPrefix(line, "## ") {
			if found {
				break
			}
			found = strings.HasPrefix(line, heading)
			continue
		}
		if found {
			section = append(section, line)
		}
	}
	return strings.TrimSpace(strings.Join(section, "\n"))
}
