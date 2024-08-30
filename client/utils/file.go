package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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
		if header.Name == tarPath {
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

func InstallArtifact(aliasGzFilePath string, installPath, artifactPath string) error {
	data, err := ReadFileFromTarGz(aliasGzFilePath, fmt.Sprintf("./%s", strings.TrimPrefix(artifactPath, string(os.PathSeparator))))
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("empty file %s", artifactPath)
	}
	localArtifactPath := filepath.Join(installPath, ResolvePath(artifactPath))
	artifactDir := filepath.Dir(localArtifactPath)
	if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
		os.MkdirAll(artifactDir, 0700)
	}
	err = os.WriteFile(localArtifactPath, data, 0700)
	if err != nil {
		return err
	}
	return nil
}
