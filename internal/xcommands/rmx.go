package xcommands

import (
	"fmt"
	"os"
	"path/filepath"
)

// RmxCommand implements the rmx command for removing files and directories
type RmxCommand struct{}

// Execute implements the Command interface
func (r *RmxCommand) Execute(args []string, flags []string) error {
	if len(args) == 0 {
		return fmt.Errorf("rmx: missing operand")
	}

	// Parse flags (dashes already stripped by parseFlags)
	recursive := false
	force := false
	for _, flag := range flags {
		switch flag {
		case "r", "R":
			recursive = true
		case "f":
			force = true
		}
	}

	// Process each target for removal
	var lastError error
	for _, target := range args {
		if err := removeTarget(target, recursive, force); err != nil {
			if !force {
				// If not in force mode, return the first error encountered
				return err
			}
			// In force mode, continue processing other targets but remember the last error
			lastError = err
		}
	}

	// In force mode, return the last error (if any) after processing all targets
	return lastError
}

// removeTarget handles the removal of a single file or directory
func removeTarget(target string, recursive, force bool) error {
	// Check if target exists
	info, err := os.Lstat(target) // Use Lstat to handle symlinks properly
	if err != nil {
		if os.IsNotExist(err) {
			if force {
				// In force mode, ignore nonexistent files
				return nil
			}
			return fmt.Errorf("rmx: cannot remove '%s': No such file or directory", target)
		}
		return fmt.Errorf("rmx: cannot access '%s': %w", target, err)
	}

	// Handle directories
	if info.IsDir() {
		if !recursive {
			return fmt.Errorf("rmx: cannot remove '%s': Is a directory", target)
		}
		return removeDirectory(target, force)
	}

	// Handle regular files and symlinks
	return removeFile(target, force)
}

// removeFile removes a single file or symlink
func removeFile(path string, force bool) error {
	if err := os.Remove(path); err != nil {
		if force && os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("rmx: cannot remove '%s': %w", path, err)
	}
	return nil
}

// removeDirectory recursively removes a directory and all its contents
func removeDirectory(dirPath string, force bool) error {
	// Use filepath.WalkDir to traverse the directory tree in reverse order
	// We need to collect all paths first, then remove them from deepest to shallowest
	var paths []string
	
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if force && os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("rmx: error accessing '%s': %w", path, err)
		}
		paths = append(paths, path)
		return nil
	})
	
	if err != nil {
		return err
	}

	// Remove files and directories in reverse order (deepest first)
	for i := len(paths) - 1; i >= 0; i-- {
		path := paths[i]
		
		// Check if it's a directory
		info, err := os.Lstat(path)
		if err != nil {
			if force && os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("rmx: cannot access '%s': %w", path, err)
		}

		if info.IsDir() {
			// Remove empty directory
			if err := os.Remove(path); err != nil {
				if force && os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("rmx: cannot remove directory '%s': %w", path, err)
			}
		} else {
			// Remove file or symlink
			if err := os.Remove(path); err != nil {
				if force && os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("rmx: cannot remove '%s': %w", path, err)
			}
		}
	}

	return nil
}

// Auto-register the command
func init() {
	Register("rmx", &RmxCommand{})
}