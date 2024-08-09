package utils

import (
	"archive/tar"
	"bytes"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/tui"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/muesli/termenv"
	"io"
	"os"
	"path/filepath"
)

var (
	Debug     logs.Level = 10
	Warn      logs.Level = 20
	Info      logs.Level = 30
	Error     logs.Level = 40
	Important logs.Level = 50

	DefaultLogStyle = map[logs.Level]string{
		Debug:     termenv.String(tui.Rocket+"[+]").Bold().Background(tui.Blue).String() + " %s ",
		Warn:      termenv.String(tui.Zap+"[warn]").Bold().Background(tui.Yellow).String() + " %s ",
		Important: termenv.String(tui.Fire+"[*]").Bold().Background(tui.Purple).String() + " %s ",
		Info:      termenv.String(tui.HotSpring+"[i]").Bold().Background(tui.Green).String() + " %s ",
		Error:     termenv.String(tui.Monster+"[-]").Bold().Background(tui.Red).String() + " %s ",
	}
)

// DeflateBuf - Deflate a buffer using BestCompression (9)
func DeflateBuf(data []byte) []byte {
	var buf bytes.Buffer
	flateWriter, _ := flate.NewWriter(&buf, flate.BestCompression)
	flateWriter.Write(data)
	flateWriter.Close()
	return buf.Bytes()
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

// ByteCountBinary - Pretty print byte size
func ByteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
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

// From the x/exp source code - gets a slice of keys for a map
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}

	return r
}
