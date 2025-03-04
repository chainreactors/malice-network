package output

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/dustin/go-humanize"
)

func NewUploadContext(content []byte) (*UploadContext, error) {
	upload := &UploadContext{}
	err := json.Unmarshal(content, upload)
	if err != nil {
		return nil, err
	}
	return upload, nil
}

type UploadContext struct {
	*FileDescriptor `json:",inline"`
	Content         []byte
}

func (u *UploadContext) Type() string {
	return consts.ContextUpload
}

func (u *UploadContext) Marshal() []byte {
	marshal, err := json.Marshal(u)
	if err != nil {
		return nil
	}
	return marshal
}

func (u *UploadContext) String() string {
	return fmt.Sprintf("Upload: %s (Size: %s)", u.Name, humanize.Bytes(uint64(u.Size)))
}

type DownloadContext struct {
	*FileDescriptor `json:",inline"`
	Content         []byte
}

func (d *DownloadContext) Type() string {
	return consts.ContextDownload
}

func (d *DownloadContext) Marshal() []byte {
	marshal, err := json.Marshal(d.FileDescriptor)
	if err != nil {
		return nil
	}
	return marshal
}

func (d *DownloadContext) String() string {
	return fmt.Sprintf("Download: %s (Size: %s )", d.Name, humanize.Bytes(uint64(d.Size)))
}
