package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/pflag"

	"github.com/go-task/task/v3/errors"
)

const (
	changelogSource = "CHANGELOG.md"
	changelogTarget = "website/src/docs/changelog.md"
	versionFile     = "internal/version/version.txt"
)

var changelogReleaseRegex = regexp.MustCompile(`## Unreleased`)

// Flags
var (
	versionFlag bool
)

func init() {
	pflag.BoolVarP(&versionFlag, "version", "v", false, "resolved version number")
	pflag.Parse()
}

func main() {
	if err := release(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func release() error {
	if len(pflag.Args()) != 1 {
		return errors.New("error: expected version number")
	}

	version, err := getVersion(versionFile)
	if err != nil {
		return err
	}

	if err := bumpVersion(version, pflag.Arg(0)); err != nil {
		return err
	}

	if versionFlag {
		fmt.Println(version)
		return nil
	}

	if err := changelog(version); err != nil {
		return err
	}

	if err := setVersionFile(versionFile, version); err != nil {
		return err
	}

	return nil
}

func getVersion(filename string) (*semver.Version, error) {
	b, err := os.ReadFile(filename)
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

	// Wrap the changelog content with v-pre directive for VitePress to prevent
	// Vue from interpreting template syntax like {{.TASK_VERSION}}
	changelogWithVPre := strings.Replace(changelog, "# Changelog\n\n", "# Changelog\n\n::: v-pre\n\n", 1) + "\n:::"

	// Add the frontmatter to the changelog
	changelogWithFrontmatter := fmt.Sprintf("---\n%s\n---\n\n%s", frontmatter, changelogWithVPre)

	// Write the changelog to the target file
	return os.WriteFile(changelogTarget, []byte(changelogWithFrontmatter), 0o644)
}

func setVersionFile(fileName string, version *semver.Version) error {
	return os.WriteFile(fileName, []byte(version.String()+"\n"), 0o644)
}
