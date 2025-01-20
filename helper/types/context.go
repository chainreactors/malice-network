package types

import (
	"encoding/json"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
)

func NewContext(typ string, content []byte) (Context, error) {
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

func NewScreenShot(content []byte) (*ScreenShot, error) {
	screenShot := &ScreenShot{}
	err := json.Unmarshal(content, screenShot)
	if err != nil {
		return nil, err
	}
	return screenShot, nil
}

func NewCredential(content []byte) (*Credential, error) {
	credential := &Credential{}
	err := json.Unmarshal(content, credential)
	if err != nil {
		return nil, err
	}
	return credential, nil
}

func NewKeyLogger(content []byte) (*KeyLogger, error) {
	keyLogger := &KeyLogger{}
	err := json.Unmarshal(content, keyLogger)
	if err != nil {
		return nil, err
	}
	return keyLogger, nil
}

type ScreenShot struct {
	FilePath string
	Content  string `json:"-"`
}

func (s *ScreenShot) Type() string {
	return consts.ContextScreenShot
}

func (s *ScreenShot) String() string {
	marshal, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(marshal)
}

type Credential struct {
	CredentialType string `json:"type"`
	Params         map[string]string
}

func (c *Credential) Type() string {
	return consts.ContextCredential
}

func (c *Credential) String() string {
	marshal, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(marshal)
}

type KeyLogger struct {
	FilePath string
	Content  string `json:"-"`
}

func (k *KeyLogger) Type() string {
	return consts.ContextKeyLogger
}

func (k *KeyLogger) String() string {
	marshal, err := json.Marshal(k)
	if err != nil {
		return ""
	}
	return string(marshal)
}

func NewPivoting(content []byte) (*Pivoting, error) {
	pivoting := &Pivoting{}
	err := json.Unmarshal(content, pivoting)
	if err != nil {
		return nil, err
	}
	return pivoting, nil
}

func NewPivotingFromProto(agent *clientpb.REMAgent) *Pivoting {
	return &Pivoting{REMAgent: agent}
}

type Pivoting struct {
	*clientpb.REMAgent `json:",inline"`
}

func (p *Pivoting) Type() string {
	return consts.ContextPivoting
}

func (p *Pivoting) String() string {
	marshal, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(marshal)
}
