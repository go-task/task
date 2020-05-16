// Copyright (c) 2017, Andrey Nering <andrey.nering@gmail.com>
// See LICENSE for licensing information

package interp

import (
	"fmt"
	"os"
)

func mkfifo(path string, mode uint32) error {
	return fmt.Errorf("unsupported")
}

// hasPermissionToDir is a no-op on Windows.
func hasPermissionToDir(info os.FileInfo) bool {
	return true
}
