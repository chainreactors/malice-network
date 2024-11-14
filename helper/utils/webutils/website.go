package webutils

import (
	"errors"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"io"
	"net/http"
	"os"
)

const (
	fileSampleSize  = 512
	defaultMimeType = "application/octet-stream"
)

func WebAddFile(web *clientpb.WebsiteAddContent, webpath string, contentType string, contentPath string,
	encryType string, parser string) error {
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

	web.Contents[webpath] = &clientpb.WebContent{
		Name:        web.Name,
		Path:        webpath,
		ContentType: contentType,
		Content:     data,
		Type:        encryType,
		Parser:      parser,
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
