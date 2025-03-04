package output

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
)

func NewPivoting(content []byte) (*PivotingContext, error) {
	pivoting := &PivotingContext{}
	err := json.Unmarshal(content, pivoting)
	if err != nil {
		return nil, err
	}
	return pivoting, nil
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

func (p *PivotingContext) String() string {
	if p.Mod == "reverse" {
		return fmt.Sprintf("Pivoting %s: %s %s <- %s on %s", p.RemID, p.Mod, p.LocalURL, p.RemoteURL, p.Pipeline)
	} else if p.Mod == "proxy" {
		return fmt.Sprintf("Pivoting %s: %s %s -> %s on %s", p.RemID, p.Mod, p.LocalURL, p.RemoteURL, p.Pipeline)
	} else {
		return string(p.Marshal())
	}
}
