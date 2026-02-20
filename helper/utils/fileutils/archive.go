package fileutils

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ArchiveFormat represents a supported archive type.
type ArchiveFormat int

const (
	ArchiveUnknown ArchiveFormat = iota
	ArchiveTarGz
	ArchiveZip
)

// DetectArchiveFormat detects archive format by extension first, then magic bytes.
func DetectArchiveFormat(path string) (ArchiveFormat, error) {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return ArchiveTarGz, nil
	case strings.HasSuffix(lower, ".zip"):
		return ArchiveZip, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return ArchiveUnknown, err
	}
	defer f.Close()

	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return ArchiveUnknown, fmt.Errorf("unsupported archive format: %s", path)
	}

	switch {
	case magic[0] == 0x1f && magic[1] == 0x8b:
		return ArchiveTarGz, nil
	case magic[0] == 0x50 && magic[1] == 0x4b && magic[2] == 0x03 && magic[3] == 0x04:
		return ArchiveZip, nil
	}
	return ArchiveUnknown, fmt.Errorf("unsupported archive format: %s", path)
}

// ReadFileFromTarGz - Read a file from a tar.gz file in-memory
func ReadFileFromTarGz(tarGzFile string, tarPath string) ([]byte, error) {
	f, err := os.Open(tarGzFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gzf, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzf.Close()

	tarReader := tar.NewReader(gzf)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		tarPath = filepath.ToSlash(tarPath)
		currentPath := strings.TrimPrefix(filepath.ToSlash(header.Name), "./")
		if currentPath == tarPath {
			switch header.Typeflag {
			case tar.TypeDir: // = directory
				continue
			case tar.TypeReg: // = regular file
				return io.ReadAll(tarReader)
			}
		}
	}
	return nil, nil
}

// ExtractTarGz extracts a .tar.gz file to the specified destination directory
func ExtractTarGz(gzipPath string, dest string) error {
	file, err := os.Open(gzipPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()

	tarReader := tar.NewReader(gz)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// ReadFileFromZip reads a single file from a zip archive by path
func ReadFileFromZip(zipFile string, targetPath string) ([]byte, error) {
	data, err := os.ReadFile(zipFile)
	if err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	targetPath = filepath.ToSlash(targetPath)
	for _, f := range zipReader.File {
		name := strings.TrimPrefix(filepath.ToSlash(f.Name), "./")
		if name == targetPath {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, nil
}

// ExtractZipFromFile extracts a zip file from disk to the destination directory
func ExtractZipFromFile(zipPath string, dest string) error {
	data, err := os.ReadFile(zipPath)
	if err != nil {
		return err
	}
	return ExtractZip(data, dest)
}

// ZIP 相关功能

// ExtractZip 解压zip文件到目标目录
func ExtractZip(zipData []byte, targetDir string) error {
	return extractZipInternal(zipData, targetDir, "", nil)
}

// ExtractZipSubdir 解压指定子目录到目标路径
func ExtractZipSubdir(zipData []byte, subdir, targetDir string) error {
	return extractZipInternal(zipData, targetDir, subdir, nil)
}

// ExtractZipWithFilter 使用过滤器解压文件
func ExtractZipWithFilter(zipData []byte, targetDir string, filter func(string) bool) error {
	return extractZipInternal(zipData, targetDir, "", filter)
}

// extractZipInternal 内部解压实现
func extractZipInternal(zipData []byte, targetDir, subdir string, filter func(string) bool) error {
	if len(zipData) == 0 {
		return fmt.Errorf("empty zip data")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// 检查子目录是否存在
	if subdir != "" && !zipHasSubdir(zipReader, subdir) {
		subdir = ""
	}

	// 解压文件
	for _, file := range zipReader.File {
		// 子目录过滤
		fileName := file.Name
		if subdir != "" {
			if !strings.HasPrefix(fileName, subdir+"/") {
				continue
			}
			fileName = strings.TrimPrefix(fileName, subdir+"/")
			if fileName == "" {
				continue
			}
		}

		// 文件过滤器
		if filter != nil && !filter(file.Name) {
			continue
		}

		outputPath := filepath.Join(targetDir, fileName)
		if err := extractZipFile(file, outputPath); err != nil {
			return fmt.Errorf("failed to extract %s: %w", file.Name, err)
		}
	}
	return nil
}

// UnzipOneWithBytes 从zip字节数据中解压单个文件并返回其内容
func UnzipOneWithBytes(content []byte) ([]byte, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("error opening ZIP file: %v", err)
	}
	if len(zipReader.File) > 1 {
		return nil, fmt.Errorf("error: multiple files in zip")
	}
	file, err := zipReader.File[0].Open()
	if err != nil {
		return nil, fmt.Errorf("error opening file inside ZIP: %v", err)
	}
	defer file.Close()
	return io.ReadAll(file)
}

// CompressFilesZip 将多个文件压缩成zip
func CompressFilesZip(filePaths []string) ([]byte, error) {
	if len(filePaths) == 0 {
		return nil, nil
	}
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	for _, filePath := range filePaths {
		if !Exist(filePath) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
		}
		defer file.Close()

		zipEntry, err := zipWriter.Create(filepath.Base(filePath))
		if err != nil {
			return nil, fmt.Errorf("failed to create zip entry for %s: %w", filePath, err)
		}

		_, err = io.Copy(zipEntry, file)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file %s to zip: %w", filePath, err)
		}
	}
	err := zipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}
	return buf.Bytes(), nil
}

// DecompressBase64ToFiles 将base64字符串解压并提取文件到指定目录
func DecompressBase64ToFiles(base64Data, outputDir string) error {
	if base64Data == "" {
		return fmt.Errorf("empty base64 data")
	}

	zipData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return ExtractZip([]byte(base64Data), outputDir)
	}

	return ExtractZip(zipData, outputDir)
}

// ZIP 辅助函数

func zipHasSubdir(zipReader *zip.Reader, subdir string) bool {
	for _, file := range zipReader.File {
		if strings.HasPrefix(file.Name, subdir+"/") {
			return true
		}
	}
	return false
}

func extractZipFile(file *zip.File, outputPath string) error {
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

func DecompressZipSubdirToRoot(zipData []byte, subdir, outputDir string) error {
	return ExtractZipSubdir(zipData, subdir, outputDir)
}
