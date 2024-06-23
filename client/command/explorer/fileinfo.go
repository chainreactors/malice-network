package explorer

import (
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"os"
	"syscall"
	"time"
)

// ProtobufDirEntry
type ProtobufDirEntry struct {
	FileInfo *implantpb.FileInfo
}

func (p ProtobufDirEntry) Name() string {
	return p.FileInfo.Name
}

func (p ProtobufDirEntry) IsDir() bool {
	return p.FileInfo.IsDir
}

func (p ProtobufDirEntry) Type() os.FileMode {
	return os.FileMode(p.FileInfo.Mode).Type()
}

func (p ProtobufDirEntry) Info() (os.FileInfo, error) {
	return p, nil
}

func (p ProtobufDirEntry) Size() int64 {
	return int64(p.FileInfo.Size)
}

func (p ProtobufDirEntry) Mode() os.FileMode {
	return os.FileMode(p.FileInfo.Mode)
}

func (p ProtobufDirEntry) ModTime() time.Time {
	return time.Unix(p.FileInfo.ModTime, 0)
}

func (p ProtobufDirEntry) Sys() interface{} {
	return &syscall.Win32FileAttributeData{}
}
