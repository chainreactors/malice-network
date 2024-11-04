package file

import "strings"

func FormatWindowPath(path string) string {
	path = strings.ReplaceAll(path, "\\\\", "/")
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.ReplaceAll(path, "/", `\`)
	return path
}
