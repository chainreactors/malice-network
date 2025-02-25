package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/parsers"
)

var (
	UPCredential    = "UP"
	TOKENCredential = "token"
	CERTCredential  = "cert"
)

func ParseZombie(content []byte) ([]*CredentialContext, error) {
	var res []*CredentialContext
	for _, b := range bytes.Split(content, []byte{'\n'}) {
		var r *parsers.ZombieResult
		if len(b) == 0 {
			continue
		}
		err := json.Unmarshal(b, &r)
		if err != nil {
			return nil, err
		}
		res = append(res, &CredentialContext{
			Target:         r.URI(),
			CredentialType: UPCredential,
			Params: map[string]string{
				"username": r.Username,
				"password": r.Password,
			},
		})
	}
	return res, nil
}

func NewCredential(content []byte) (*CredentialContext, error) {
	credential := &CredentialContext{}
	err := json.Unmarshal(content, credential)
	if err != nil {
		return nil, err
	}
	return credential, nil
}

type CredentialContext struct {
	CredentialType string            `json:"type"`
	Target         string            `json:"target"`
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
	return fmt.Sprintf("%s: %s %s\n", c.CredentialType, c.Target, MapJoin(c.Params))
}

func MapJoin(m map[string]string) string {
	var s string
	for k, v := range m {
		s += fmt.Sprintf("%s: %s ", k, v)
	}
	return s
}
