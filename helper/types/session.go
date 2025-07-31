package types

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"

	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
)

func NewSessionContext(req *clientpb.RegisterSession) *SessionContext {
	return &SessionContext{
		SessionInfo: &SessionInfo{
			Os:       &implantpb.Os{},
			Process:  &implantpb.Process{},
			ProxyURL: req.RegisterData.Proxy,
			Interval: req.RegisterData.Timer.Interval,
			Jitter:   req.RegisterData.Timer.Jitter,
		},
		KeyPair: nil, // 密钥对在后续初始化时设置
		Modules: req.RegisterData.Module,
		Addons:  req.RegisterData.Addons,
		Argue:   map[string]string{},
		Any:     map[string]interface{}{},
	}
}

func RecoverSessionContext(content string) (*SessionContext, error) {
	var sessionContext *SessionContext
	err := json.Unmarshal([]byte(content), &sessionContext)
	if err != nil {
		return nil, err
	}
	return sessionContext, nil
}

type SessionContext struct {
	*SessionInfo `json:",inline"`
	KeyPair      *clientpb.KeyPair      `json:"key_pair,omitempty"` // Age 密钥对
	Modules      []string               `json:"modules"`
	Addons       []*implantpb.Addon     `json:"addons"`
	Argue        map[string]string      `json:"argue"` // 参数欺骗
	Any          map[string]interface{} `json:"any"`
}

func (ctx *SessionContext) Data() map[string]interface{} {
	result := make(map[string]interface{})
	err := mapstructure.Decode(ctx, &result)
	if err != nil {
		return nil
	}
	return result
}

func (ctx *SessionContext) Marshal() string {
	data, _ := json.Marshal(ctx)
	return string(data)
}

func (ctx *SessionContext) Update(req *clientpb.RegisterSession) {
	ctx.Modules = req.RegisterData.Module
	ctx.Addons = req.RegisterData.Addons
}

func (ctx *SessionContext) GetAny(id string) (interface{}, bool) {
	v, ok := ctx.Any[id]
	return v, ok
}

type SessionInfo struct {
	Os          *implantpb.Os      `json:"os"`
	Process     *implantpb.Process `json:"process"`
	Interval    uint64             `json:"interval"`
	Jitter      float64            `json:"jitter"`
	IsPrivilege bool               `json:"is_privilege"`
	Filepath    string             `json:"filepath"`
	WorkDir     string             `json:"workdir"`
	ProxyURL    string             `json:"proxy"`
	Locale      string             `json:"locale"`
}

// IsSecureEnabled 检查是否启用了安全模式
func (ctx *SessionContext) IsSecureEnabled() bool {
	return ctx.KeyPair != nil && ctx.KeyPair.PublicKey != "" && ctx.KeyPair.PrivateKey != ""
}

// GetKeyPair 返回当前的密钥对信息（用于传输到 listener）
func (ctx *SessionContext) GetKeyPair() *clientpb.KeyPair {
	return ctx.KeyPair
}

// SetKeyPair 设置密钥对信息（从数据库恢复或新生成时）
func (ctx *SessionContext) SetKeyPair(keyPair *clientpb.KeyPair) {
	ctx.KeyPair = keyPair
}
