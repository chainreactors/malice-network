package fileutils

import "path/filepath"

func FormatWindowPath(path string) string {
	return filepath.FromSlash(path)
}
