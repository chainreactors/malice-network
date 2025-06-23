package intl

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed community/* professional/* custom/*
var UnifiedFS embed.FS

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

// ParseEmbedPath 解析embed://路径，返回level和文件路径
func ParseEmbedPath(embedPath string) (levelName, filePath string, err error) {
	if !strings.HasPrefix(embedPath, "embed://") {
		return "", "", fmt.Errorf("invalid embed path: %s", embedPath)
	}

	// 移除embed://前缀
	path := strings.TrimPrefix(embedPath, "embed://")

	// 分割为level和文件路径
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid embed path format: %s", embedPath)
	}

	return parts[0], parts[1], nil
}

// ReadEmbedResource 根据embed://路径读取嵌入式资源
func ReadEmbedResource(embedPath string) ([]byte, error) {
	levelName, filePath, err := ParseEmbedPath(embedPath)
	if err != nil {
		return nil, err
	}

	// 构建完整路径
	fullPath := levelName + "/" + filePath
	return GetFileContent(fullPath)
}

// ListLevels 列出所有可用的级别（community, professional, custom）
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
