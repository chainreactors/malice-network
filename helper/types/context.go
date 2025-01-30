package types

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/parsers"
	"strings"
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
	jsonMarshal, err := json.Marshal(file)
	if err != nil {
		return "", err
	}
	return string(jsonMarshal), nil
}

// AsContext 将Context接口转换为具体的实现类型
func AsContext[T Context](ctx Context) (T, error) {
	if t, ok := ctx.(T); ok {
		return t, nil
	}
	var zero T
	return zero, fmt.Errorf("cannot convert %T to %T", ctx, zero)
}

func ToContext[T Context](ctx *clientpb.Context) (T, error) {
	c, err := ParseContext(ctx.Type, ctx.Value)
	if err != nil {
		var zero T
		return zero, err
	}
	return AsContext[T](c)
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
	case consts.ContextPort:
		ctx, err = NewPortContext(content)

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
	// Marshal 返回用于存储到数据库的序列化数据，忽略大型二进制数据
	Marshal() []byte
	// String 返回context的简要描述
	String() string
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
	return fmt.Sprintf("Download: %s (Size: %.2f KB)", d.Name, float64(d.Size)/1024)
}

type ScreenShotContext struct {
	*FileDescriptor `json:",inline"`
	Content         []byte
}

func (s *ScreenShotContext) Type() string {
	return consts.ContextScreenShot
}

func (s *ScreenShotContext) Marshal() []byte {
	marshal, err := json.Marshal(s.FileDescriptor)
	if err != nil {
		return nil
	}
	return marshal
}

func (s *ScreenShotContext) String() string {
	return fmt.Sprintf("Screenshot: %s (Size: %.2f KB)", s.Name, float64(s.Size)/1024)
}

type KeyLoggerContext struct {
	*FileDescriptor `json:",inline"`
	Content         []byte
}

func (k *KeyLoggerContext) Type() string {
	return consts.ContextKeyLogger
}

func (k *KeyLoggerContext) Marshal() []byte {
	marshal, err := json.Marshal(k.FileDescriptor)
	if err != nil {
		return nil
	}
	return marshal
}

func (k *KeyLoggerContext) String() string {
	return fmt.Sprintf("Keylogger: %s (Size: %.2f KB)", k.Name, float64(k.Size)/1024)
}

type CredentialContext struct {
	CredentialType string            `json:"type"`
	Params         map[string]string `json:"params"`
}

func (c *CredentialContext) Type() string {
	return consts.ContextCredential
}

func (c *CredentialContext) Marshal() []byte {
	marshal, err := json.Marshal(c)
	if err != nil {
		return nil
	}
	return marshal
}

func (c *CredentialContext) String() string {
	return fmt.Sprintf("Credential[%s]: %s", c.CredentialType, c.Params["username"])
}

func NewDownloadContext(content []byte) (*DownloadContext, error) {
	downloadContext := &DownloadContext{}
	err := json.Unmarshal(content, downloadContext)
	if err != nil {
		return nil, err
	}
	return downloadContext, nil
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

func NewScreenShot(content []byte) (*ScreenShotContext, error) {
	screenShot := &FileDescriptor{}
	err := json.Unmarshal(content, screenShot)
	if err != nil {
		return nil, err
	}
	return &ScreenShotContext{FileDescriptor: screenShot}, nil
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

func (p *PivotingContext) Marshal() []byte {
	marshal, err := json.Marshal(p)
	if err != nil {
		return nil
	}
	return marshal
}

func (p *PivotingContext) String() string {
	return fmt.Sprintf("Pivoting: %s -> %s", p.GetLocal(), p.GetRemote())
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
	return fmt.Sprintf("Upload: %s (Size: %.2f KB)", u.Name, float64(u.Size)/1024)
}

type Port struct {
	Ip       string `json:"ip"`
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
	Status   string `json:"status"`
}

func NewPortContext(content []byte) (*PortContext, error) {
	portContext := &PortContext{}
	err := json.Unmarshal(content, portContext)
	if err != nil {
		return nil, err
	}
	return portContext, nil
}

type PortContext struct {
	Ports   []*Port     `json:"ports"`
	Extends interface{} `json:"extend"`
}

func (p *PortContext) Type() string {
	return consts.ContextPort
}

func (p *PortContext) Marshal() []byte {
	marshal, err := json.Marshal(p)
	if err != nil {
		return nil
	}
	return marshal
}

func (p *PortContext) GogoData() (*parsers.GOGOData, bool) {
	data, ok := p.Extends.(*parsers.GOGOData)
	return data, ok
}

func (p *PortContext) String() string {
	var ports []string
	for _, port := range p.Ports {
		ports = append(ports, fmt.Sprintf("%s:%s/%s", port.Ip, port.Port, port.Protocol))
	}
	return fmt.Sprintf("Ports: %s", strings.Join(ports, ", "))
}
