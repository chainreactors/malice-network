package types

import (
	"encoding/json"
	"fmt"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

type FileDescriptor struct {
	Name       string `json:"name"`
	TargetPath string `json:"target_path"`
	FilePath   string `json:"filepath"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"`
	Abstract   string `json:"abstract"`
}

func (file *FileDescriptor) Marshal() (string, error) {
	jsonString, err := json.Marshal(file)
	if err != nil {
		return "", err
	}
	return string(jsonString), nil
}

// AsContext 将Context接口转换为具体的实现类型
func AsContext[T Context](ctx Context) (T, error) {
	if t, ok := ctx.(T); ok {
		return t, nil
	}
	var zero T
	return zero, fmt.Errorf("cannot convert %T to %T", ctx, zero)
}

func ParseContext(typ string, content []byte) (Context, error) {
	var ctx Context
	var err error
	switch typ {
	case consts.ContextScreenShot:
		ctx, err = NewScreenShot(content)
	case consts.ContextCredential:
		ctx, err = NewCredential(content)
	case consts.ContextKeyLogger:
		ctx, err = NewKeyLogger(content)
	case consts.ContextPivoting:
		ctx, err = NewPivoting(content)
	case consts.ContextDownload:
		ctx, err = NewDownloadContext(content)
	case consts.ContextUpload:
		ctx, err = NewUploadContext(content)
	}
	return ctx, err
}

func MarshalContext(ctx Context) []byte {
	marshal, err := json.Marshal(ctx)
	if err != nil {
		return nil
	}
	return marshal
}

type Context interface {
	Type() string
	String() string
}

func NewDownloadContext(content []byte) (*DownloadContext, error) {
	downloadContext := &DownloadContext{}
	err := json.Unmarshal(content, downloadContext)
	if err != nil {
		return nil, err
	}
	return downloadContext, nil
}

type DownloadContext struct {
	*FileDescriptor `json:",inline"`
	Content         []byte `json:"-"`
}

func (d *DownloadContext) Type() string {
	return consts.ContextDownload
}

func (d *DownloadContext) String() string {
	marshal, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	return string(marshal)
}

func NewScreenShot(content []byte) (*ScreenShotContext, error) {
	screenShot := &ScreenShotContext{}
	err := json.Unmarshal(content, screenShot)
	if err != nil {
		return nil, err
	}
	return screenShot, nil
}

func NewCredential(content []byte) (*CredentialContext, error) {
	credential := &CredentialContext{}
	err := json.Unmarshal(content, credential)
	if err != nil {
		return nil, err
	}
	return credential, nil
}

func NewKeyLogger(content []byte) (*KeyLoggerContext, error) {
	keyLogger := &KeyLoggerContext{}
	err := json.Unmarshal(content, keyLogger)
	if err != nil {
		return nil, err
	}
	return keyLogger, nil
}

type ScreenShotContext struct {
	*FileDescriptor `json:",inline"`
	Content         []byte `json:"-"`
}

func (s *ScreenShotContext) Type() string {
	return consts.ContextScreenShot
}

func (s *ScreenShotContext) String() string {
	marshal, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(marshal)
}

type CredentialContext struct {
	CredentialType string `json:"type"`
	Params         map[string]string
}

func (c *CredentialContext) Type() string {
	return consts.ContextCredential
}

func (c *CredentialContext) String() string {
	marshal, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(marshal)
}

type KeyLoggerContext struct {
	*FileDescriptor `json:",inline"`
	Content         []byte `json:"-"`
}

func (k *KeyLoggerContext) Type() string {
	return consts.ContextKeyLogger
}

func (k *KeyLoggerContext) String() string {
	marshal, err := json.Marshal(k)
	if err != nil {
		return ""
	}
	return string(marshal)
}

func NewPivoting(content []byte) (*PivotingContext, error) {
	pivoting := &PivotingContext{}
	err := json.Unmarshal(content, pivoting)
	if err != nil {
		return nil, err
	}
	return pivoting, nil
}

func NewPivotingFromProto(agent *clientpb.REMAgent) *PivotingContext {
	return &PivotingContext{REMAgent: agent}
}

type PivotingContext struct {
	*clientpb.REMAgent `json:",inline"`
}

func (p *PivotingContext) Type() string {
	return consts.ContextPivoting
}

func (p *PivotingContext) String() string {
	marshal, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(marshal)
}

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
}

func (u *UploadContext) Type() string {
	return consts.ContextUpload
}

func (u *UploadContext) String() string {
	marshal, err := json.Marshal(u)
	if err != nil {
		return ""
	}
	return string(marshal)
}
