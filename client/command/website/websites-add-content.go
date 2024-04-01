package website

import (
	"context"
	"errors"
	"github.com/AlecAivazis/survey/v2"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func websiteAddCmd(c *grumble.Context, con *console.Console) {
	cPath := c.Flags.String("content-path")
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
	addWeb := &clientpb.WebsiteAddContent{
		Name:     name,
		Contents: map[string]*clientpb.WebContent{},
	}

	if fileIfo.IsDir() {
		if !recursive && !confirmAddDirectory() {
			return
		}
		webAddDirectory(addWeb, webPath, cPath)
	} else {
		webAddFile(addWeb, webPath, contentType, cPath)
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

func webAddDirectory(web *clientpb.WebsiteAddContent, webpath string, contentPath string) {
	fullLocalPath, _ := filepath.Abs(contentPath)
	filepath.Walk(contentPath, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {

			// localPath is the full absolute path to the file, so we cut it down
			fullWebpath := path.Join(webpath, filepath.ToSlash(localPath[len(fullLocalPath):]))
			webAddFile(web, fullWebpath, "", localPath)
		}
		return nil
	})
}

func webAddFile(web *clientpb.WebsiteAddContent, webpath string, contentType string, contentPath string) error {
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

	web.Contents[webpath] = &clientpb.WebContent{
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

func confirmAddDirectory() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "Recursively add entire directory?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}
