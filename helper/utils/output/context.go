package output

import (
	"encoding/json"
	"fmt"
	"strings"

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
	jsonMarshal, err := json.Marshal(file)
	if err != nil {
		return "", err
	}
	return string(jsonMarshal), nil
}

type Context interface {
	Type() string
	// Marshal 返回用于存储到数据库的序列化数据，忽略大型二进制数据
	Marshal() []byte
	// String 返回context的简要描述
	String() string
}

// AsContext 将Context接口转换为具体的实现类型
func AsContext[T Context](ctx Context) (T, error) {
	if t, ok := ctx.(T); ok {
		return t, nil
	}
	var zero T
	return zero, fmt.Errorf("cannot convert %T to %T", ctx, zero)
}

func AsContexts[T Context](ctxs []Context) ([]T, error) {
	var contexts []T
	for _, ctx := range ctxs {
		c, err := AsContext[T](ctx)
		if err != nil {
			return nil, err
		}
		contexts = append(contexts, c)
	}
	return contexts, nil
}

func ToContext[T Context](ctx *clientpb.Context) (T, error) {
	c, err := ParseContext(ctx.Type, ctx.Value)
	if err != nil {
		var zero T
		return zero, err
	}
	return AsContext[T](c)
}

func ToContexts[T Context](ctxs []*clientpb.Context) ([]T, error) {
	var contexts []T
	for _, ctx := range ctxs {
		c, err := ToContext[T](ctx)
		if err != nil {
			return nil, err
		}
		contexts = append(contexts, c)
	}
	return contexts, nil
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

type Contexts []Context

func (ctxs Contexts) String() string {
	var s strings.Builder
	for _, ctx := range ctxs {
		s.WriteString(ctx.Type() + "\n")
		s.WriteString(ctx.String() + "\n")
	}
	return s.String()
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
	return fmt.Sprintf("%s (Size: %.2f KB)", s.Name, float64(s.Size)/1024)
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

func NewDownloadContext(content []byte) (*DownloadContext, error) {
	downloadContext := &DownloadContext{}
	err := json.Unmarshal(content, downloadContext)
	if err != nil {
		return nil, err
	}
	return downloadContext, nil
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
