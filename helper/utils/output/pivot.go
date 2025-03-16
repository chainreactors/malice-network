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
		Enable:     true,
		Pipeline:   agent.PipelineId,
		RemAgentID: agent.Id,
		Mod:        agent.Mod,
		RemoteURL:  agent.Remote,
		LocalURL:   agent.Local,
	}
}

type PivotingContext struct {
	Enable     bool   `json:"enable"`
	Listener   string `json:"listener_id"`
	Pipeline   string `json:"pipeline"`
	RemAgentID string `json:"id"`
	LocalURL   string `json:"local"`
	RemoteURL  string `json:"remote"`
	Mod        string `json:"mod"`
}

func (p *PivotingContext) ToRemAgent() *clientpb.REMAgent {
	return &clientpb.REMAgent{
		Id:         p.RemAgentID,
		PipelineId: p.Pipeline,
		Mod:        p.Mod,
		Local:      p.LocalURL,
		Remote:     p.RemoteURL,
		Enable:     p.Enable,
	}
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
		return fmt.Sprintf("%s serving %s", p.RemAgentID, p.RemoteURL)
	} else if p.Mod == "proxy" {
		return fmt.Sprintf("%s serving %s", p.RemAgentID, p.LocalURL)
	} else if p.Mod == "connect" {
		return fmt.Sprintf("%s connecting to %s", p.RemAgentID, p.Pipeline)
	} else {
		return fmt.Sprintf("invalid mod %s", p.Mod)
	}
}

func (p *PivotingContext) String() string {
	if p.Mod == "reverse" {
		return fmt.Sprintf("Pivoting %s: %s %s <- %s on %s", p.RemAgentID, p.Mod, p.LocalURL, p.RemoteURL, p.Pipeline)
	} else if p.Mod == "proxy" {
		return fmt.Sprintf("Pivoting %s: %s %s -> %s on %s", p.RemAgentID, p.Mod, p.LocalURL, p.RemoteURL, p.Pipeline)
	} else if p.Mod == "connect" {
		return fmt.Sprintf("Pivoting %s: %s connected on %s", p.RemAgentID, p.Mod, p.Pipeline)
	} else {
		return string(p.Marshal())
	}
}
