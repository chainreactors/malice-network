package file

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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
