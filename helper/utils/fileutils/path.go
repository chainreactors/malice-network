package fileutils

import (
	"path"
	"regexp"
	"strings"
)

func FormatWindowPath(path string) string {
	return strings.ReplaceAll(path, "/", `\`)
}

func CheckWindowsPath(path string) bool {
	paths := strings.Split(path, "\\")

	if len(paths) < 2 {
		return false
	}

	for i, p := range paths {
		if p == "" {
			return false
		}
		if len(p) > 224 {
			return false
		}

		if i > 0 {
			if strings.Contains(p, ":") {
				return false
			}
			if strings.TrimSpace(p) != p {
				return false
			}
			if strings.Contains(p, "/") {
				return false
			}
		} else {
			if !strings.Contains(p, ":") || strings.Contains(p, " ") {
				return false
			}
			if len(strings.Split(p, ":")) != 2 {
				return false
			}
			matched, _ := regexp.MatchString("[a-zA-Z]", strings.Split(p, ":")[0])
			if !matched {
				return false
			}
		}
	}

	invalidChars := []string{"?", "/", "|", "<", ">", "*", `"`}
	for _, char := range invalidChars {
		if strings.Contains(path, char) {
			return false
		}
	}
	return true
}

func CheckLinuxPath(path string) bool {
	if path == "" || path[0] != '/' {
		return false
	}

	paths := strings.Split(path, "/")
	if len(paths) < 2 {
		return false
	}

	for i, p := range paths {
		if i != 0 && (p == "" || strings.TrimSpace(p) != p) {
			return false
		}
	}

	invalidChars := []string{"?", "|", "<", ">", "*", `"`, ":"}
	for _, char := range invalidChars {
		if strings.Contains(path, char) {
			return false
		}
	}
	return true
}
// toForwardSlash normalises a remote path (which may use Windows backslashes)
// to forward slashes so that the standard path package works on any OS.
func toForwardSlash(p string) string {
	return strings.ReplaceAll(p, `\`, "/")
}

// RemoteBase extracts the last element of a remote path that may use either
// forward or backward slashes, working correctly on any host OS.
func RemoteBase(p string) string {
	return path.Base(toForwardSlash(p))
}

// RemoteDir returns the parent directory of a remote path.
func RemoteDir(p string) string {
	return path.Dir(toForwardSlash(p))
}

// RemoteJoin joins remote path elements using forward slashes,
// suitable for building implant-side paths that work across OSes.
func RemoteJoin(elem ...string) string {
	return path.Join(elem...)
}

func CheckFullPath(path string) bool {
	checkWindowsPath := CheckWindowsPath(path)
	checkLinuxPath := CheckLinuxPath(path)
	return checkWindowsPath || checkLinuxPath
}
