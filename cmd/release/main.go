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
	changelogTarget = "docs/docs/changelog.mdx"
)

var (
	changelogReleaseRegex = regexp.MustCompile(`## Unreleased`)
	versionRegex          = regexp.MustCompile(`(?m)^  "version": "\d+\.\d+\.\d+",$`)
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

	if err := setJSONVersion("package.json", version); err != nil {
		return err
	}

	if err := setJSONVersion("package-lock.json", version); err != nil {
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
	// Open changelog target file
	b, err := os.ReadFile(changelogTarget)
	if err != nil {
		return err
	}

	// Get the current frontmatter
	currentChangelog := string(b)
	sections := strings.SplitN(currentChangelog, "---", 3)
	if len(sections) != 3 {
		return errors.New("error: invalid frontmatter")
	}
	frontmatter := strings.TrimSpace(sections[1])

	// Open changelog source file
	b, err = os.ReadFile(changelogSource)
	if err != nil {
		return err
	}
	changelog := string(b)
	date := time.Now().Format("2006-01-02")

	// Replace "Unreleased" with the new version and date
	changelog = changelogReleaseRegex.ReplaceAllString(changelog, fmt.Sprintf("## v%s - %s", version, date))

	// Write the changelog to the source file
	if err := os.WriteFile(changelogSource, []byte(changelog), 0o644); err != nil {
		return err
	}

	// Add the frontmatter to the changelog
	changelog = fmt.Sprintf("---\n%s\n---\n\n%s", frontmatter, changelog)

	// Write the changelog to the target file
	return os.WriteFile(changelogTarget, []byte(changelog), 0o644)
}

func setJSONVersion(fileName string, version *semver.Version) error {
	// Read the JSON file
	b, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	// Replace the version
	new := versionRegex.ReplaceAllString(string(b), fmt.Sprintf(`  "version": "%s",`, version.String()))

	// Write the JSON file
	return os.WriteFile(fileName, []byte(new), 0o644)
}
