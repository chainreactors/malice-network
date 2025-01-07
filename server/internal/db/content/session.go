package content

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
)

func NewSessionContext(req *clientpb.RegisterSession) *SessionContext {
	return &SessionContext{
		SessionInfo: &SessionInfo{
			ProxyURL: req.RegisterData.Proxy,
			Interval: req.RegisterData.Timer.Interval,
			Jitter:   req.RegisterData.Timer.Jitter,
		},
		Modules: req.RegisterData.Module,
		Addons:  req.RegisterData.Addons,
		Loot:    map[string]string{},
		Argue:   map[string]string{},
		Data:    map[string]string{},
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
	*SessionInfo
	Modules []string
	Addons  []*implantpb.Addon
	Loot    map[string]string // mimikatz,zombie
	Argue   map[string]string // 参数欺骗
	Data    map[string]string // 元数据
	Any     map[string]interface{}
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
	Os          *implantpb.Os
	Process     *implantpb.Process
	Interval    uint64
	Jitter      float64
	IsPrivilege bool
	Filepath    string
	WordDir     string
	ProxyURL    string
	Locale      string
}
