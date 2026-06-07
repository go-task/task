package fingerprint

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-task/task/v3/internal/gitignore"
)

type gitignoreRule struct {
	dir     string
	matcher gitignore.Matcher
}

// loadGitignoreRules walks up from dir collecting .gitignore files.
// Stops at the first .git (file or directory) found.
// Returns nil if no .git is found (not in a git repo).
func loadGitignoreRules(dir string) []gitignoreRule {
	dir, _ = filepath.Abs(dir)

	var rules []gitignoreRule
	foundGit := false
	current := dir

	for {
		lines := readGitignoreLines(filepath.Join(current, ".gitignore"))
		if len(lines) > 0 {
			patterns := make([]gitignore.Pattern, 0, len(lines))
			for _, line := range lines {
				patterns = append(patterns, gitignore.ParsePattern(line, nil))
			}
			rules = append(rules, gitignoreRule{
				dir:     current,
				matcher: gitignore.NewMatcher(patterns),
			})
		}
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			foundGit = true
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	if !foundGit {
		return nil
	}

	return rules
}

func readGitignoreLines(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil
	}
	return lines
}

// filterGitignored removes entries from the file map that match gitignore rules.
func filterGitignored(files map[string]bool, dir string) map[string]bool {
	rules := loadGitignoreRules(dir)
	if len(rules) == 0 {
		return files
	}

	for path := range files {
		for _, rule := range rules {
			relPath, err := filepath.Rel(rule.dir, path)
			if err != nil || strings.HasPrefix(relPath, "..") {
				continue
			}
			// Sources are files, not directories; pass isDir=false. Per the
			// gitignore spec this still matches files under an ignored dir
			// (e.g. "build/" matches build/out.txt).
			if rule.matcher.Match(strings.Split(filepath.ToSlash(relPath), "/"), false) {
				files[path] = false
				break
			}
		}
	}

	return files
}
