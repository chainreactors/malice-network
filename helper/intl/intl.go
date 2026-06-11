package intl

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// GetFileContent 获取嵌入式文件内容
func GetFileContent(filename string) ([]byte, error) {
	content, err := UnifiedFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded file %s: %w", filename, err)
	}
	return content, nil
}

// FileExists 检查嵌入式文件是否存在
func FileExists(filename string) bool {
	_, err := UnifiedFS.Open(filename)
	return err == nil
}

// ReadDir 读取嵌入式目录内容
func ReadDir(dirname string) ([]fs.DirEntry, error) {
	return UnifiedFS.ReadDir(dirname)
}

// ListLevelPlugins 列出指定级别下的多插件目录（包含 mal.yaml 的一级子目录）
func ListLevelPlugins(levelName string) ([]string, error) {
	entries, err := ReadDir(levelName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	pluginNames := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := levelName + "/" + entry.Name() + "/mal.yaml"
		if _, err := UnifiedFS.Open(manifestPath); err == nil {
			pluginNames = append(pluginNames, entry.Name())
		}
	}

	sort.Strings(pluginNames)
	return pluginNames, nil
}

// FindResource 查找嵌入式资源文件，支持架构特定查找
func FindResource(levelName, base, arch, ext string) ([]byte, error) {
	// 构建资源文件路径
	filename := fmt.Sprintf("%s/%s_%s.%s", levelName, base, arch, ext)

	// 先尝试精确匹配
	if content, err := GetFileContent("resources/" + filename); err == nil {
		return content, nil
	}

	// 如果精确匹配失败，尝试不带架构的版本
	fallbackFilename := fmt.Sprintf("%s/%s.%s", levelName, base, ext)
	if content, err := GetFileContent("resources/" + fallbackFilename); err == nil {
		return content, nil
	}

	return nil, fmt.Errorf("resource not found: %s", filename)
}

// GetResourcePath 获取嵌入式资源的embed://路径格式
func GetResourcePath(levelName, filename string) string {
	return fmt.Sprintf("embed://%s/resources/%s", levelName, filename)
}

// ParseEmbedPath 解析embed://路径，返回命名空间和文件路径
func ParseEmbedPath(embedPath string) (namespace, filePath string, err error) {
	if !strings.HasPrefix(embedPath, "embed://") {
		return "", "", fmt.Errorf("invalid embed path: %s", embedPath)
	}

	path := strings.Trim(strings.TrimPrefix(embedPath, "embed://"), "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid embed path format: %s", embedPath)
	}

	return strings.TrimSpace(parts[0]), strings.TrimPrefix(parts[1], "/"), nil
}

func isKnownLevel(namespace string) bool {
	switch namespace {
	case "custom", "professional", "community":
		return true
	default:
		return false
	}
}

func buildFallbackEmbedCandidates(namespace, filePath string) []string {
	levels := []string{"custom", "professional", "community"}
	seen := make(map[string]struct{}, len(levels)+1)
	candidates := make([]string, 0, len(levels)+1)

	add := func(path string) {
		path = strings.Trim(path, "/")
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		candidates = append(candidates, path)
	}

	normalizedFilePath := strings.Trim(filePath, "/")
	for {
		prefix := namespace + "/"
		if !strings.HasPrefix(normalizedFilePath, prefix) {
			break
		}
		normalizedFilePath = strings.TrimPrefix(normalizedFilePath, prefix)
	}

	add(namespace + "/" + normalizedFilePath)
	for _, level := range levels {
		add(level + "/" + filePath)
		add(level + "/" + normalizedFilePath)
		add(level + "/" + namespace + "/" + filePath)
		add(level + "/" + namespace + "/" + normalizedFilePath)
	}

	if isKnownLevel(namespace) {
		add(namespace + "/" + filePath)
		add(namespace + "/" + namespace + "/" + filePath)
		add(namespace + "/" + namespace + "/" + normalizedFilePath)
	}

	return candidates
}

// ReadEmbedResource 根据embed://路径读取嵌入式资源
func ReadEmbedResource(embedPath string) ([]byte, error) {
	namespace, filePath, err := ParseEmbedPath(embedPath)
	if err != nil {
		return nil, err
	}

	directPath := strings.Trim(strings.TrimPrefix(embedPath, "embed://"), "/")
	if content, readErr := GetFileContent(directPath); readErr == nil {
		return content, nil
	}

	for _, candidate := range buildFallbackEmbedCandidates(namespace, filePath) {
		if content, readErr := GetFileContent(candidate); readErr == nil {
			return content, nil
		}
	}

	return nil, fmt.Errorf("embedded resource not found: %s", embedPath)
}

// ListLevels 列出所有可用的级别（community, professional）
func ListLevels() ([]string, error) {
	entries, err := ReadDir(".")
	if err != nil {
		return nil, err
	}

	var levels []string
	for _, entry := range entries {
		if entry.IsDir() {
			levels = append(levels, entry.Name())
		}
	}

	return levels, nil
}

// GetLevelManifest 获取指定级别的manifest文件内容
func GetLevelManifest(levelName string) ([]byte, error) {
	manifestPath := levelName + "/mal.yaml"
	return GetFileContent(manifestPath)
}

// GetLevelEntryFile 获取指定级别的入口文件内容
func GetLevelEntryFile(levelName, entryFile string) ([]byte, error) {
	entryPath := levelName + "/" + entryFile
	return GetFileContent(entryPath)
}

// GetLevelResource 获取指定级别的资源文件内容
func GetLevelResource(levelName, resourceName string) ([]byte, error) {
	resourcePath := levelName + "/resources/" + resourceName
	return GetFileContent(resourcePath)
}
