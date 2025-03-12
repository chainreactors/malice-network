package output

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

func NewPivoting(content []byte) (*PivotingContext, error) {
	pivoting := &PivotingContext{}
	err := json.Unmarshal(content, pivoting)
	if err != nil {
		return nil, err
	}
	return pivoting, nil
}

func NewPivotingWithRem(agent *clientpb.REMAgent) *PivotingContext {
	return &PivotingContext{
		Enable:    true,
		Pipeline:  agent.PipelineId,
		RemID:     agent.Id,
		Mod:       agent.Mod,
		RemoteURL: agent.Remote,
		LocalURL:  agent.Local,
	}
}

type PivotingContext struct {
	Enable    bool   `json:"enable"`
	Listener  string `json:"listener_id"`
	Pipeline  string `json:"pipeline"`
	RemID     string `json:"id"`
	LocalURL  string `json:"local"`
	RemoteURL string `json:"remote"`
	Mod       string `json:"mod"`
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

func (p *PivotingContext) Abstract() string {
	if p.Mod == "reverse" {
		return fmt.Sprintf("%s serving %s", p.RemID, p.RemoteURL)
	} else if p.Mod == "proxy" {
		return fmt.Sprintf("%s serving %s", p.RemID, p.LocalURL)
	} else if p.Mod == "connect" {
		return fmt.Sprintf("%s connecting to %s", p.RemID, p.Pipeline)
	} else {
		return fmt.Sprintf("invalid mod %s", p.Mod)
	}
}

func (p *PivotingContext) String() string {
	if p.Mod == "reverse" {
		return fmt.Sprintf("Pivoting %s: %s %s <- %s on %s", p.RemID, p.Mod, p.LocalURL, p.RemoteURL, p.Pipeline)
	} else if p.Mod == "proxy" {
		return fmt.Sprintf("Pivoting %s: %s %s -> %s on %s", p.RemID, p.Mod, p.LocalURL, p.RemoteURL, p.Pipeline)
	} else {
		return string(p.Marshal())
	}
}
