package xcommands

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// CpCommand implements the cp command
type CpCommand struct{}

// Execute implements the Command interface
func (c *CpCommand) Execute(args []string, flags []string) error {
	if len(args) < 2 {
		return fmt.Errorf("cp: missing operands")
	}

	source := args[len(args)-2]
	dest := args[len(args)-1]

	// Handle flags (dashes already stripped)
	recursive := false
	preserve := false
	for _, flag := range flags {
		switch flag {
		case "r", "R":
			recursive = true
		case "p":
			preserve = true
		}
	}

	return copyFile(source, dest, recursive, preserve)
}

// copyFile handles the actual file copying logic
func copyFile(source, dest string, recursive, preserve bool) error {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("cp: cannot stat '%s': %w", source, err)
	}

	if sourceInfo.IsDir() {
		if !recursive {
			return fmt.Errorf("cp: '%s' is a directory (not copied)", source)
		}
		return copyDirectory(source, dest, preserve)
	}

	return copySingleFile(source, dest, preserve)
}

// copySingleFile copies a single file
func copySingleFile(source, dest string, preserve bool) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("cp: cannot open '%s': %w", source, err)
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("cp: cannot stat '%s': %w", source, err)
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("cp: cannot create directory '%s': %w", destDir, err)
	}

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("cp: cannot create '%s': %w", dest, err)
	}
	defer destFile.Close()

	// Copy file contents
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("cp: error copying '%s' to '%s': %w", source, dest, err)
	}

	// Preserve permissions and timestamps if requested
	if preserve {
		if err := destFile.Chmod(sourceInfo.Mode()); err != nil {
			return fmt.Errorf("cp: cannot preserve permissions for '%s': %w", dest, err)
		}
		if err := os.Chtimes(dest, sourceInfo.ModTime(), sourceInfo.ModTime()); err != nil {
			return fmt.Errorf("cp: cannot preserve timestamps for '%s': %w", dest, err)
		}
	} else {
		// Set basic permissions
		if err := destFile.Chmod(sourceInfo.Mode()); err != nil {
			return fmt.Errorf("cp: cannot set permissions for '%s': %w", dest, err)
		}
	}

	return nil
}

// copyDirectory recursively copies a directory
func copyDirectory(source, dest string, preserve bool) error {
	// Create destination directory
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("cp: cannot create directory '%s': %w", dest, err)
	}

	// Walk through source directory
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from source
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("cp: cannot determine relative path: %w", err)
		}

		destPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			// Create directory
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("cp: cannot create directory '%s': %w", destPath, err)
			}

			// Preserve directory permissions if requested
			if preserve {
				info, err := d.Info()
				if err != nil {
					return fmt.Errorf("cp: cannot get info for '%s': %w", path, err)
				}
				if err := os.Chmod(destPath, info.Mode()); err != nil {
					return fmt.Errorf("cp: cannot preserve permissions for '%s': %w", destPath, err)
				}
			}
		} else {
			// Copy file
			if err := copySingleFile(path, destPath, preserve); err != nil {
				return err
			}
		}

		return nil
	})
}

// Auto-register the command
func init() {
	Register("cpx", &CpCommand{})
}