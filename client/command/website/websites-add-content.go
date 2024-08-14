package website

import (
	"context"
	"errors"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/tui"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func websiteAddCmd(c *grumble.Context, con *console.Console) {
	cPath := c.Args.String("content-path")
	webPath := c.Flags.String("web-path")
	name := c.Flags.String("name")
	contentType := c.Flags.String("content-type")
	recursive := c.Flags.Bool("recursive")
	if name == "" {
		console.Log.Errorf("Must specify a website name via --name, see --help")
		return
	}
	if webPath == "" {
		console.Log.Errorf("Must specify a web path via --path, see --help")
		return
	}
	if cPath == "" {
		console.Log.Errorf("Must specify some --content-path\n")
		return
	}
	cPath, _ = filepath.Abs(cPath)

	fileIfo, err := os.Stat(cPath)
	if err != nil {
		console.Log.Errorf("Error adding content %s\n", err)
		return
	}
	addWeb := &lispb.WebsiteAddContent{
		Name:     name,
		Contents: map[string]*lispb.WebContent{},
	}

	if fileIfo.IsDir() {
		if !recursive && !ConfirmAddDirectory() {
			return
		}
		WebAddDirectory(addWeb, webPath, cPath)
	} else {
		WebAddFile(addWeb, webPath, contentType, cPath)
	}
	_, err = con.Rpc.WebsiteAddContent(context.Background(), addWeb)
	if err != nil {
		console.Log.Errorf("%s", err)
		return
	}
	console.Log.Importantf("Content added to website %s", name)
	// TODO - PrintWebsite(web, con)
	return
}

func WebAddDirectory(web *lispb.WebsiteAddContent, webpath string, contentPath string) {
	fullLocalPath, _ := filepath.Abs(contentPath)
	filepath.Walk(contentPath, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {

			// localPath is the full absolute path to the file, so we cut it down
			fullWebpath := path.Join(webpath, filepath.ToSlash(localPath[len(fullLocalPath):]))
			WebAddFile(web, fullWebpath, "", localPath)
		}
		return nil
	})
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
		contentType = sniffContentType(file)
	}

	web.Contents[webpath] = &lispb.WebContent{
		Path:        webpath,
		ContentType: contentType,
		Content:     data,
	}
	return nil
}

func sniffContentType(out *os.File) string {
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

func ConfirmAddDirectory() bool {
	confirmModel := tui.NewConfirm("Recursively add entire directory?")
	newConfirm := tui.NewModel(confirmModel, nil, false, true)
	err := newConfirm.Run()
	if err != nil {
		console.Log.Errorf("Error running confirm model: %s", err)
		return false
	}
	return confirmModel.Confirmed
}
