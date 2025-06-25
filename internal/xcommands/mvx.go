package xcommands

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// MvxCommand implements the mv command
type MvxCommand struct{}

// Execute implements the Command interface
func (m *MvxCommand) Execute(args []string, flags []string) error {
	if len(args) < 2 {
		return fmt.Errorf("mvx: missing operands")
	}

	source := args[len(args)-2]
	dest := args[len(args)-1]

	// Handle flags (dashes already stripped)
	force := false
	interactive := false
	for _, flag := range flags {
		switch flag {
		case "f", "force":
			force = true
		case "i", "interactive":
			interactive = true
		}
	}

	return moveFile(source, dest, force, interactive)
}

// moveFile handles the actual file moving logic
func moveFile(source, dest string, force, interactive bool) error {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("mvx: cannot stat '%s': %w", source, err)
	}

	// Check if destination exists
	destInfo, destExists := os.Stat(dest)
	if destExists == nil {
		// Destination exists, handle accordingly
		if destInfo.IsDir() && sourceInfo.IsDir() {
			// Moving directory into directory
			dest = filepath.Join(dest, filepath.Base(source))
		} else if destInfo.IsDir() && !sourceInfo.IsDir() {
			// Moving file into directory
			dest = filepath.Join(dest, filepath.Base(source))
		} else if !destInfo.IsDir() && !force && !interactive {
			// Destination file exists and no force/interactive flag
			return fmt.Errorf("mvx: cannot move '%s' to '%s': file exists", source, dest)
		}
		
		// Check again if the final destination exists after path adjustment
		if _, err := os.Stat(dest); err == nil && !force && !interactive {
			return fmt.Errorf("mvx: cannot move '%s' to '%s': file exists", source, dest)
		}
	}

	// Try atomic rename first (works if source and dest are on same filesystem)
	if err := os.Rename(source, dest); err == nil {
		return nil
	}

	// If rename fails (likely cross-filesystem), fall back to copy + remove
	if sourceInfo.IsDir() {
		return moveDirectory(source, dest)
	}

	return moveSingleFile(source, dest)
}

// moveSingleFile moves a single file by copying then removing original
func moveSingleFile(source, dest string) error {
	// First copy the file
	if err := copySingleFileForMove(source, dest); err != nil {
		return err
	}

	// Then remove the original
	if err := os.Remove(source); err != nil {
		// If removal fails, try to clean up the destination
		os.Remove(dest)
		return fmt.Errorf("mvx: cannot remove '%s': %w", source, err)
	}

	return nil
}

// copySingleFileForMove copies a single file (helper for cross-filesystem moves)
func copySingleFileForMove(source, dest string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("mvx: cannot open '%s': %w", source, err)
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("mvx: cannot stat '%s': %w", source, err)
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("mvx: cannot create directory '%s': %w", destDir, err)
	}

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("mvx: cannot create '%s': %w", dest, err)
	}
	defer destFile.Close()

	// Copy file contents
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("mvx: error copying '%s' to '%s': %w", source, dest, err)
	}

	// Preserve permissions and timestamps
	if err := destFile.Chmod(sourceInfo.Mode()); err != nil {
		return fmt.Errorf("mvx: cannot preserve permissions for '%s': %w", dest, err)
	}

	if err := os.Chtimes(dest, sourceInfo.ModTime(), sourceInfo.ModTime()); err != nil {
		return fmt.Errorf("mvx: cannot preserve timestamps for '%s': %w", dest, err)
	}

	return nil
}

// moveDirectory moves a directory by copying then removing original
func moveDirectory(source, dest string) error {
	// First copy the entire directory structure
	if err := copyDirectoryForMove(source, dest); err != nil {
		return err
	}

	// Then remove the original directory
	if err := os.RemoveAll(source); err != nil {
		// If removal fails, try to clean up the destination
		os.RemoveAll(dest)
		return fmt.Errorf("mvx: cannot remove directory '%s': %w", source, err)
	}

	return nil
}

// copyDirectoryForMove recursively copies a directory (helper for cross-filesystem moves)
func copyDirectoryForMove(source, dest string) error {
	// Get source directory info for permissions
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("mvx: cannot stat '%s': %w", source, err)
	}

	// Create destination directory with source permissions
	if err := os.MkdirAll(dest, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("mvx: cannot create directory '%s': %w", dest, err)
	}

	// Walk through source directory
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory since we already created it
		if path == source {
			return nil
		}

		// Calculate relative path from source
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("mvx: cannot determine relative path: %w", err)
		}

		destPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			// Create directory with preserved permissions
			info, err := d.Info()
			if err != nil {
				return fmt.Errorf("mvx: cannot get info for '%s': %w", path, err)
			}

			if err := os.MkdirAll(destPath, info.Mode()); err != nil {
				return fmt.Errorf("mvx: cannot create directory '%s': %w", destPath, err)
			}

			// Preserve timestamps
			if err := os.Chtimes(destPath, info.ModTime(), info.ModTime()); err != nil {
				return fmt.Errorf("mvx: cannot preserve timestamps for '%s': %w", destPath, err)
			}
		} else {
			// Copy file
			if err := copySingleFileForMove(path, destPath); err != nil {
				return err
			}
		}

		return nil
	})
}

// Auto-register the command
func init() {
	Register("mvx", &MvxCommand{})
}