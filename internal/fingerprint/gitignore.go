package fingerprint

import (
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// filterGitignored removes entries from the file map that match gitignore rules.
// Files are expected to be absolute paths. The dir parameter is used to find the git repository.
// Returns the input map unchanged if the directory is not inside a git repository.
func filterGitignored(files map[string]bool, dir string) map[string]bool {
	repo, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return files
	}

	wt, err := repo.Worktree()
	if err != nil {
		return files
	}

	var allPatterns []gitignore.Pattern

	if ps, err := gitignore.LoadSystemPatterns(wt.Filesystem); err == nil {
		allPatterns = append(allPatterns, ps...)
	}

	if ps, err := gitignore.LoadGlobalPatterns(wt.Filesystem); err == nil {
		allPatterns = append(allPatterns, ps...)
	}

	if ps, err := gitignore.ReadPatterns(wt.Filesystem, nil); err == nil {
		allPatterns = append(allPatterns, ps...)
	}

	if len(allPatterns) == 0 {
		return files
	}

	matcher := gitignore.NewMatcher(allPatterns)
	gitRoot := wt.Filesystem.Root()

	for path := range files {
		relPath, err := filepath.Rel(gitRoot, path)
		if err != nil {
			continue
		}
		pathComponents := strings.Split(filepath.ToSlash(relPath), "/")
		if matcher.Match(pathComponents, false) {
			files[path] = false
		}
	}

	return files
}
