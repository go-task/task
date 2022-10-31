//go:build windows

package sysinfo

// NOTE: This always returns -1 since there is currently no easy way to get
// file owner information on Windows.
func Owner(path string) (int, error) {
	return -1, nil
}
