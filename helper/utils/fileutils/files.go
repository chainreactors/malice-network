package fileutils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/klauspost/compress/flate"
	"io"
	"os"
	"path/filepath"
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

func ForceRemoveAll(rootPath string) {
	ChmodR(rootPath, 0600, 0700)
	os.RemoveAll(rootPath)
}

func MoveFile(sourcePath, destPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

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
	sourceFile.Close()
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
