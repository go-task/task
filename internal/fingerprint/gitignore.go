package fingerprint

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-task/task/v3/internal/gitignore"
)

type linesCacheEntry struct {
	mtime time.Time
	lines []string
}

var (
	gitignoreLinesCache sync.Map // dir -> linesCacheEntry, invalidated by mtime
	repoRootCache       sync.Map // dir -> repo root (or "" when not in a repo)
)

// findRepoRoot returns the first ancestor of dir containing a .git entry, or
// ("", false) when dir is not inside a git repository.
func findRepoRoot(dir string) (string, bool) {
	if v, ok := repoRootCache.Load(dir); ok {
		root := v.(string)
		return root, root != ""
	}

	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			repoRootCache.Store(dir, current)
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	repoRootCache.Store(dir, "")
	return "", false
}

// filterGitignored marks entries matching gitignore rules as excluded (false).
// All .gitignore files from the repo root down to each candidate file's
// directory feed a single matcher so that precedence and cross-file negations
// (`!pattern`) resolve correctly.
func filterGitignored(files map[string]bool, dir string) map[string]bool {
	if len(files) == 0 {
		return files
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return files
	}
	repoRoot, ok := findRepoRoot(absDir)
	if !ok {
		return files
	}

	// Every directory from the repo root down to each candidate file's dir, so
	// nested .gitignore files reached by deep globs are included too.
	dirSet := make(map[string]struct{})
	for path, included := range files {
		if !included {
			continue
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		d := filepath.Dir(absPath)
		if !withinRepo(repoRoot, d) {
			continue
		}
		for {
			dirSet[d] = struct{}{}
			if d == repoRoot {
				break
			}
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}

	// Shallow dirs first (lower priority): the matcher scans patterns last to
	// first, so deeper rules win and can negate shallower ones.
	dirs := make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool {
		di := strings.Count(dirs[i], string(filepath.Separator))
		dj := strings.Count(dirs[j], string(filepath.Separator))
		if di != dj {
			return di < dj
		}
		return dirs[i] < dirs[j]
	})

	var patterns []gitignore.Pattern
	for _, d := range dirs {
		lines := readGitignoreLines(d)
		if len(lines) == 0 {
			continue
		}
		// domain scopes each pattern to its .gitignore subtree (go-git semantics).
		var domain []string
		if rel, err := filepath.Rel(repoRoot, d); err == nil && rel != "." {
			domain = strings.Split(filepath.ToSlash(rel), "/")
		}
		for _, line := range lines {
			patterns = append(patterns, gitignore.ParsePattern(line, domain))
		}
	}
	if len(patterns) == 0 {
		return files
	}

	matcher := gitignore.NewMatcher(patterns)
	for path, included := range files {
		if !included {
			continue
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		relPath, err := filepath.Rel(repoRoot, absPath)
		if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
			continue
		}
		if ignored(matcher, strings.Split(filepath.ToSlash(relPath), "/")) {
			files[path] = false
		}
	}

	return files
}

// ignored honors Git's rule that a file under an ignored directory cannot be
// re-included by a deeper negation: if any ancestor directory is ignored, so is
// the file. Otherwise the file's own verdict applies (isDir=false).
func ignored(matcher gitignore.Matcher, segments []string) bool {
	for i := 1; i < len(segments); i++ {
		if matcher.Match(segments[:i], true) {
			return true
		}
	}
	return matcher.Match(segments, false)
}

func withinRepo(repoRoot, p string) bool {
	rel, err := filepath.Rel(repoRoot, p)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// readGitignoreLines returns the .gitignore lines in dir (nil if none), cached
// per directory and invalidated by mtime so watch mode picks up edits.
func readGitignoreLines(dir string) []string {
	path := filepath.Join(dir, ".gitignore")
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	mtime := info.ModTime()

	if v, ok := gitignoreLinesCache.Load(dir); ok {
		entry := v.(linesCacheEntry)
		if entry.mtime.Equal(mtime) {
			return entry.lines
		}
	}

	lines := parseGitignoreLines(path)
	gitignoreLinesCache.Store(dir, linesCacheEntry{mtime: mtime, lines: lines})
	return lines
}

func parseGitignoreLines(path string) []string {
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
	// On a scan error (e.g. an over-long line) keep what was parsed rather than
	// dropping the whole file.
	return lines
}
