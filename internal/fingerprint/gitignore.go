package fingerprint

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

type gitignoreRule struct {
	dir     string
	matcher *ignore.GitIgnore
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
			rules = append(rules, gitignoreRule{
				dir:     current,
				matcher: ignore.CompileIgnoreLines(lines...),
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
		line := scanner.Text()
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
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
			if rule.matcher.MatchesPath(filepath.ToSlash(relPath)) {
				files[path] = false
				break
			}
		}
	}

	return files
}
