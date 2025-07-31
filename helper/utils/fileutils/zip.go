package fileutils

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZipExtractor 提供优雅的 ZIP 解压功能
type ZipExtractor struct {
	zipData []byte
}

// NewZipExtractor 创建新的 ZIP 解压器
func NewZipExtractor(zipData []byte) *ZipExtractor {
	return &ZipExtractor{zipData: zipData}
}

// ExtractSubdir 解压指定子目录到目标路径
func (z *ZipExtractor) ExtractSubdir(subdir, targetDir string) error {
	if len(z.zipData) == 0 {
		return fmt.Errorf("empty zip data")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(z.zipData), int64(len(z.zipData)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// 检查子目录是否存在
	if !z.hasSubdir(zipReader, subdir) {
		subdir = ""
	}

	return z.extractFiles(zipReader, subdir, targetDir)
}

// ExtractAll 解压所有文件到目标路径
func (z *ZipExtractor) ExtractAll(targetDir string) error {
	if len(z.zipData) == 0 {
		return fmt.Errorf("empty zip data")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(z.zipData), int64(len(z.zipData)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	return z.extractFiles(zipReader, "", targetDir)
}

// ExtractWithFilter 使用过滤器解压文件
func (z *ZipExtractor) ExtractWithFilter(targetDir string, filter func(string) bool) error {
	if len(z.zipData) == 0 {
		return fmt.Errorf("empty zip data")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(z.zipData), int64(len(z.zipData)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	return z.extractFilesWithFilter(zipReader, targetDir, filter)
}

// hasSubdir 检查 ZIP 中是否存在指定子目录
func (z *ZipExtractor) hasSubdir(zipReader *zip.Reader, subdir string) bool {
	for _, file := range zipReader.File {
		if strings.HasPrefix(file.Name, subdir+"/") {
			return true
		}
	}
	return false
}

// extractFiles 提取文件的核心逻辑
func (z *ZipExtractor) extractFiles(zipReader *zip.Reader, subdir, targetDir string) error {
	for _, file := range zipReader.File {
		// 如果指定了子目录，只处理该子目录下的文件
		if subdir != "" {
			if !strings.HasPrefix(file.Name, subdir+"/") {
				continue
			}
			// 移除子目录前缀
			file.Name = strings.TrimPrefix(file.Name, subdir+"/")
			if file.Name == "" {
				continue // 跳过子目录本身
			}
		}

		outputPath := filepath.Join(targetDir, file.Name)

		if err := z.extractFile(file, outputPath); err != nil {
			return fmt.Errorf("failed to extract %s: %w", file.Name, err)
		}
	}
	return nil
}

// extractFilesWithFilter 使用过滤器提取文件
func (z *ZipExtractor) extractFilesWithFilter(zipReader *zip.Reader, targetDir string, filter func(string) bool) error {
	for _, file := range zipReader.File {
		if !filter(file.Name) {
			continue
		}

		outputPath := filepath.Join(targetDir, file.Name)
		if err := z.extractFile(file, outputPath); err != nil {
			return fmt.Errorf("failed to extract %s: %w", file.Name, err)
		}
	}
	return nil
}

// extractFile 提取单个文件
func (z *ZipExtractor) extractFile(file *zip.File, outputPath string) error {
	if file.FileInfo().IsDir() {
		return os.MkdirAll(outputPath, 0755)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip entry: %w", err)
	}
	defer rc.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return os.Chmod(outputPath, file.Mode())
}

// CompressFilesToBase64 将多个文件压缩成zip并转换为base64字符串
func CompressFilesToBase64(filePaths []string) (string, error) {
	if len(filePaths) == 0 {
		return "", nil
	}
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	for _, filePath := range filePaths {
		if !Exist(filePath) {
			return "", fmt.Errorf("file not found: %s", filePath)
		}
		file, err := os.Open(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
		}
		defer file.Close()
		_, err = file.Stat()
		if err != nil {
			return "", fmt.Errorf("failed to get file info for %s: %w", filePath, err)
		}
		zipEntry, err := zipWriter.Create(filepath.Base(filePath))
		if err != nil {
			return "", fmt.Errorf("failed to create zip entry for %s: %w", filePath, err)
		}

		_, err = io.Copy(zipEntry, file)
		if err != nil {
			return "", fmt.Errorf("failed to copy file %s to zip: %w", filePath, err)
		}
	}
	err := zipWriter.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close zip writer: %w", err)
	}
	zipData := buf.Bytes()
	base64Data := base64.StdEncoding.EncodeToString(zipData)
	return base64Data, nil
}

// DecompressBase64ToFiles 将base64字符串解压并提取文件到指定目录
func DecompressBase64ToFiles(zipData, outputDir string) error {
	if zipData == "" {
		return fmt.Errorf("empty zip data")
	}

	// 创建zip reader
	zipReader, err := zip.NewReader(bytes.NewReader([]byte(zipData)), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 遍历zip文件中的所有条目
	for _, file := range zipReader.File {
		// 构建输出文件路径
		outputPath := filepath.Join(outputDir, file.Name)

		// 如果是目录，创建目录
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(outputPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", outputPath, err)
			}
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", outputPath, err)
		}

		// 打开zip文件条目
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip entry %s: %w", file.Name, err)
		}

		// 创建输出文件
		outFile, err := os.Create(outputPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
		}

		// 复制文件内容
		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to copy file %s: %w", file.Name, err)
		}

		// 设置文件权限
		if err := os.Chmod(outputPath, file.Mode()); err != nil {
			return fmt.Errorf("failed to set file permissions for %s: %w", outputPath, err)
		}
	}

	return nil
}

// 便捷函数，保持向后兼容
func DecompressZipSubdirToRoot(zipData []byte, subdir, outputDir string) error {
	extractor := NewZipExtractor(zipData)
	return extractor.ExtractSubdir(subdir, outputDir)
}
