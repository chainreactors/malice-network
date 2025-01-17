package types

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/helper/consts"
)

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
	return consts.ScreenShotType
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
	return consts.CredentialType
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
	return consts.KeyLoggerType
}

func (k *KeyLogger) String() string {
	marshal, err := json.Marshal(k)
	if err != nil {
		return ""
	}
	return string(marshal)
}
