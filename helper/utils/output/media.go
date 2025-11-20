package output

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/dustin/go-humanize"
)

type MediaContext struct {
	*FileDescriptor `json:",inline"`
	Identifier      string `json:"identifier"`
	MediaKind       string `json:"media_kind"`
	Content         []byte `json:"-"`
}

func (m *MediaContext) Type() string {
	return consts.ContextMedia
}

func (m *MediaContext) Marshal() []byte {
	type alias struct {
		*FileDescriptor
		Identifier string `json:"identifier"`
		MediaKind  string `json:"media_kind"`
	}
	data := alias{
		FileDescriptor: m.FileDescriptor,
		Identifier:     m.Identifier,
		MediaKind:      m.MediaKind,
	}
	bytes, err := json.Marshal(&data)
	if err != nil {
		return nil
	}
	return bytes
}

func (m *MediaContext) String() string {
	kind := m.MediaKind
	if kind == "" {
		kind = "media"
	}
	return fmt.Sprintf("%s (%s, %s)", m.Name, kind, humanize.Bytes(uint64(m.Size)))
}

func NewMediaContext(content []byte) (*MediaContext, error) {
	media := &MediaContext{}
	err := json.Unmarshal(content, media)
	if err != nil {
		return nil, err
	}
	return media, nil
}
