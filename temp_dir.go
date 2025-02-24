package task

import (
	"path/filepath"
	"strings"

	"github.com/go-task/task/v3/internal/env"
	"github.com/go-task/task/v3/internal/execext"
	"github.com/go-task/task/v3/internal/filepathext"
)

type TempDir struct {
	Remote      string
	Fingerprint string
}

func NewTempDir(dir string) (*TempDir, error) {
	tempDir, err := setupTempDirFingerprint(dir)
	if err != nil {
		return nil, err
	}

	err = setupTempDirRemote(dir, tempDir)
	if err != nil {
		return nil, err
	}

	return tempDir, nil
}

func setupTempDirFingerprint(dir string) (*TempDir, error) {
	tempDir := env.GetTaskEnv("TEMP_DIR")

	if tempDir == "" {
		return &TempDir{
			Remote:      filepathext.SmartJoin(dir, ".task"),
			Fingerprint: filepathext.SmartJoin(dir, ".task"),
		}, nil
	}

	if filepath.IsAbs(tempDir) || strings.HasPrefix(tempDir, "~") {
		tempDir, err := execext.ExpandLiteral(tempDir)
		if err != nil {
			return nil, err
		}
		projectDir, _ := filepath.Abs(dir)
		projectName := filepath.Base(projectDir)
		return &TempDir{
			Remote:      tempDir,
			Fingerprint: filepathext.SmartJoin(tempDir, projectName),
		}, nil
	}

	return &TempDir{
		Remote:      filepathext.SmartJoin(dir, tempDir),
		Fingerprint: filepathext.SmartJoin(dir, tempDir),
	}, nil
}

func setupTempDirRemote(dir string, tempDir *TempDir) error {
	remoteDir := env.GetTaskEnv("REMOTE_DIR")

	if remoteDir == "" {
		return nil
	}

	if filepath.IsAbs(remoteDir) || strings.HasPrefix(remoteDir, "~") {
		remoteTempDir, err := execext.ExpandLiteral(remoteDir)
		if err != nil {
			return err
		}
		tempDir.Remote = remoteTempDir
		return nil
	}

	tempDir.Remote = filepathext.SmartJoin(dir, ".task")
	return nil
}
