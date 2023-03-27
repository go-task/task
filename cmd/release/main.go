package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

const (
	changelogSource = "CHANGELOG.md"
	changelogTarget = "docs/docs/changelog.md"
)

const changelogTemplate = `---
slug: /changelog/
sidebar_position: 7
---`

var (
	changelogReleaseRegex = regexp.MustCompile(`## Unreleased`)
	changelogUserRegex    = regexp.MustCompile(`@(\w+)`)
	changelogIssueRegex   = regexp.MustCompile(`#(\d+)`)
)

func main() {
	if err := release(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func release() error {
	if len(os.Args) != 2 {
		return errors.New("error: expected version number")
	}

	version, err := getVersion()
	if err != nil {
		return err
	}

	if err := bumpVersion(version, os.Args[1]); err != nil {
		return err
	}

	fmt.Println(version)

	if err := changelog(version); err != nil {
		return err
	}

	return nil
}

func getVersion() (*semver.Version, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	b, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return semver.NewVersion(strings.TrimSpace(string(b)))
}

func bumpVersion(version *semver.Version, verb string) error {
	switch verb {
	case "major":
		*version = version.IncMajor()
	case "minor":
		*version = version.IncMinor()
	case "patch":
		*version = version.IncPatch()
	default:
		*version = *semver.MustParse(verb)
	}
	return nil
}

func changelog(version *semver.Version) error {
	// Open changelog source file
	b, err := os.ReadFile(changelogSource)
	if err != nil {
		panic(err)
	}
	changelog := string(b)
	date := time.Now().Format("2006-01-02")

	// Replace "Unreleased" with the new version and date
	changelog = changelogReleaseRegex.ReplaceAllString(changelog, fmt.Sprintf("## v%s - %s", version, date))

	// Write the changelog to the source file
	if err := os.WriteFile(changelogSource, []byte(changelog), 0644); err != nil {
		panic(err)
	}

	// Add the frontmatter to the changelog
	changelog = fmt.Sprintf("%s\n\n%s", changelogTemplate, changelog)

	// Replace @user and #issue with full links
	changelog = changelogUserRegex.ReplaceAllString(changelog, "[@$1](https://github.com/$1)")
	changelog = changelogIssueRegex.ReplaceAllString(changelog, "[#$1](https://github.com/go-task/task/issues/$1)")

	// Write the changelog to the target file
	if err := os.WriteFile(changelogTarget, []byte(changelog), 0644); err != nil {
		panic(err)
	}

	return nil
}
