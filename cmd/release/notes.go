package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func notes(w io.Writer) error {
	version, err := getVersion(versionFile)
	if err != nil {
		return err
	}

	b, err := os.ReadFile(changelogSource)
	if err != nil {
		return err
	}

	section := changelogSection(string(b), version)
	if section == "" {
		return nil
	}
	_, err = fmt.Fprintln(w, section)
	return err
}

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
