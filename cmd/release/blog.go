package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"go.yaml.in/yaml/v3"
)

// Validate every new post before writing any of them or promoting the website.
func dateNewBlogPosts(next, latest, date string) error {
	type update struct {
		path string
		data string
	}
	var updates []update
	err := filepath.WalkDir(next, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() == "index.md" || filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(next, path)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(latest, rel)); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
		data, err := os.ReadFile(path) //nolint:gosec // Release inputs are trusted files in the local checkout.
		if err != nil {
			return err
		}
		content, err := blogPostDate(string(data), date)
		if err != nil {
			return fmt.Errorf("%s: invalid frontmatter: %w", path, err)
		}
		if content == string(data) {
			return nil
		}
		updates = append(updates, update{path, content})
		return nil
	})
	if err != nil {
		return err
	}
	for _, update := range updates {
		if err := os.WriteFile(update.path, []byte(update.data), 0o644); err != nil { //nolint:gosec
			return err
		}
		fmt.Printf("Updated blog date: %s → %s\n", update.path, date)
	}
	return nil
}

var (
	frontmatterDelimiter = regexp.MustCompile(`(?m)^---[\t ]*\r?$`)
	// Include indented continuation lines if the previous date was a YAML block.
	blogDateField = regexp.MustCompile(`(?m)^(?:date|'date'|"date"):[^\r\n]*(?:\r?\n[\t ]+[^\r\n]*)*`)
)

func blogPostDate(data, date string) (string, error) {
	delimiters := frontmatterDelimiter.FindAllStringIndex(data, 2)
	if len(delimiters) != 2 || delimiters[0][0] != 0 {
		return "", fmt.Errorf("expected opening and closing --- delimiters")
	}
	frontmatter := data[delimiters[0][1]:delimiters[1][0]]
	var metadata map[string]any
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return "", err
	}
	if blogDateField.MatchString(frontmatter) {
		frontmatter = blogDateField.ReplaceAllString(frontmatter, "date: "+date)
	} else if _, exists := metadata["date"]; exists {
		return "", fmt.Errorf("expected date on its own unindented line")
	} else {
		frontmatter += "date: " + date + "\n"
	}
	// Reject unusual YAML layouts that cannot accept a standalone date line.
	metadata = nil
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return "", err
	}
	// Keep the closing delimiter and everything after it verbatim.
	return data[:delimiters[0][1]] + frontmatter + data[delimiters[1][0]:], nil
}
