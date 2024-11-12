//go:build !windows

package fileutils

import "path/filepath"

// ResolvePath - Resolve a path from an assumed root path
func ResolvePath(in string) string {
	return filepath.Clean("/" + in)
}
