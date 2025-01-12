package fileutils

import (
	"path/filepath"
	"regexp"
	"strings"
)

func FormatWindowPath(path string) string {
	return filepath.FromSlash(path)
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
func CheckFullPath(path string) bool {
	checkWindowsPath := CheckWindowsPath(path)
	checkLinuxPath := CheckLinuxPath(path)
	return checkWindowsPath || checkLinuxPath
}
