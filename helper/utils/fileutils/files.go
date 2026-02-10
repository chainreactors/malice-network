package fileutils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/h2non/filetype"
	"io"
	"os"
	"path/filepath"

	"encoding/base64"
	"github.com/klauspost/compress/flate"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// DeflateBuf - Deflate a buffer using BestCompression (9)
func DeflateBuf(data []byte) []byte {
	var buf bytes.Buffer
	flateWriter, _ := flate.NewWriter(&buf, flate.BestCompression)
	flateWriter.Write(data)
	flateWriter.Close()
	return buf.Bytes()
}

// CopyFile - Copy a file from src to dst
func CopyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	err = out.Close()
	if err != nil {
		return err
	}
	return err
}

// Exist - Check if a file exists
func Exist(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// checkIfDirectory - Check if a path is a directory
func checkIfDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Println("Error:", err)
		return false
	}
	return fileInfo.IsDir()
}

// RemoveFile - Remove a file from src to dst
func RemoveFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}

func CalculateSHA256Checksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	return checksum, nil
}

func CalculateSHA256Byte(data []byte) string {
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	return hashStr
}

// ChmodR - Recursively chmod
func ChmodR(path string, filePerm, dirPerm os.FileMode) error {
	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			if info.IsDir() {
				err = os.Chmod(name, dirPerm)
			} else {
				err = os.Chmod(name, filePerm)
			}
		}
		return err
	})
}

func ForceRemoveAll(rootPath string) error {
	err := ChmodR(rootPath, 0600, 0700)
	if err != nil {
		return err
	}
	return os.RemoveAll(rootPath)
}

func MoveFile(sourcePath, destPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	if err = destFile.Sync(); err != nil {
		return err
	}

	return os.Remove(sourcePath)
}

func MoveDirectory(sourceDir, destDir string) error {
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
			return err
		}
	}

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, os.ModePerm)
		} else {
			if err := MoveFile(path, destPath); err != nil {
				return fmt.Errorf("failed to move file %s to %s: %w", path, destPath, err)
			}
		}
		return nil
	})
}

func GetExtensionByPath(filepath string) (string, error) {

	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	buf := make([]byte, 261)
	_, err = file.Read(buf)
	if err != nil {
		return "", fmt.Errorf("read file error: %w", err)
	}

	return GetExtensionByBytes(buf)
}

func GetExtensionByBytes(data []byte) (string, error) {
	kind, err := filetype.Match(data)
	if err != nil {
		return "", fmt.Errorf("unknown file type %s", err)
	}
	return "." + kind.Extension, nil
}

// CopyDirectoryExcept copies all files and directories from sourceDir to targetDir except the excluded files.
func CopyDirectoryExcept(sourceDir, targetDir string, excludeFiles []string) error {
	excludeSet := make(map[string]struct{})
	for _, f := range excludeFiles {
		excludeSet[f] = struct{}{}
	}

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		if _, skip := excludeSet[info.Name()]; skip {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		targetPath := filepath.Join(targetDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}
		return CopyFile(path, targetPath)
	})
}

// DetectContentType
func DetectContentType(data []byte) string {
	if len(data) == 0 {
		return "unknown"
	}

	// 使用 filetype.Match 检测文件类型
	kind, err := filetype.Match(data)
	if err != nil {
		return "unknown"
	}
	if kind.MIME.Type == "application/zip" || kind.Extension == "zip" {
		return "zip"
	}
	if kind.MIME.Type == "text/plain" || kind.Extension == "yml" || kind.Extension == "yaml" {
		return "yaml"
	}

	return "unknown"
}

// WithTempDir executes a function with a temporary directory, automatically cleaning up
func WithTempDir(prefix string, fn func(tempDir string) error) error {
	tempDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	return fn(tempDir)
}

// CollectFilePaths collects all file paths from the given directory recursively
func CollectFilePaths(rootPath string) ([]string, error) {
	var filePaths []string
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && path != rootPath {
			filePaths = append(filePaths, path)
		}
		return nil
	})
	return filePaths, err
}

// DecodeBase64OrRaw attempts to decode base64 string, falls back to raw bytes if decoding fails
func DecodeBase64OrRaw(data string) ([]byte, error) {
	if data == "" {
		return nil, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err == nil {
		return decoded, nil
	}
	return []byte(data), nil
}

// EncodeBase64OrRaw encodes the given data to base64 string, falls back to raw bytes if encoding fails
func EncodeBase64OrRaw(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
