package types

import (
	"errors"
	"github.com/chainreactors/malice-network/helper/proto/listener/lispb"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

const (
	fileSampleSize  = 512
	defaultMimeType = "application/octet-stream"
)

func WebAddDirectory(web *lispb.WebsiteAddContent, webpath string, contentPath string) *lispb.WebsiteAssets {
	var webAssets = &lispb.WebsiteAssets{
		Assets: []*lispb.WebsiteAsset{},
	}
	fullLocalPath, _ := filepath.Abs(contentPath)
	filepath.Walk(contentPath, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// localPath is the full absolute path to the file, so we cut it down
			fullWebpath := path.Join(webpath, filepath.ToSlash(localPath[len(fullLocalPath):]))
			WebAddFile(web, fullWebpath, "", localPath)
			content, err := os.ReadFile(localPath)
			if err != nil {
				return err
			}
			webAssets.Assets = append(webAssets.Assets, &lispb.WebsiteAsset{
				WebName:  web.Name,
				Content:  content,
				FileName: fullWebpath,
			})
		}
		return nil
	})
	return webAssets
}

func WebAddFile(web *lispb.WebsiteAddContent, webpath string, contentType string, contentPath string) error {
	fileInfo, err := os.Stat(contentPath)
	if os.IsNotExist(err) {
		return err // contentPath does not exist
	}
	if fileInfo.IsDir() {
		return errors.New("file content path is directory")
	}

	file, err := os.Open(contentPath)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	if contentType == "" {
		contentType = SniffContentType(file)
	}

	web.Contents[webpath] = &lispb.WebContent{
		Name:        web.Name,
		Path:        webpath,
		ContentType: contentType,
		Content:     data,
	}
	return nil
}

func SniffContentType(out *os.File) string {
	fileInfo, err := out.Stat()
	if err != nil {
		return defaultMimeType
	}

	readSize := fileSampleSize
	if fileInfo.Size() < int64(fileSampleSize) {
		readSize = int(fileInfo.Size())
	}

	out.Seek(0, io.SeekStart)

	buffer := make([]byte, readSize)
	_, err = out.Read(buffer)
	if err != nil {
		return defaultMimeType
	}

	contentType := http.DetectContentType(buffer)
	return contentType
}
