package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/parsers"
	"strings"
)

var (
	GOGOPortType = "gogo"
)

func ParseGOGO(content []byte) (*PortContext, error) {
	var res parsers.GOGOResults
	for _, b := range bytes.Split(content, []byte{'\n'}) {
		var r *parsers.GOGOResult
		if len(b) == 0 {
			continue
		}
		err := json.Unmarshal(b, &r)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	ports := make([]*Port, 0, len(res))
	for _, r := range res {
		ports = append(ports, &Port{
			Ip:       r.Ip,
			Port:     r.Port,
			Protocol: r.Protocol,
			Status:   r.Status,
		})
	}
	return &PortContext{
		Ports: ports,
	}, nil
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
	var ports strings.Builder
	if p.Extends != nil {
		switch e := p.Extends.(type) {
		case *parsers.GOGOResults:
			for _, r := range *e {
				ports.WriteString(r.FullOutput())
			}
		}
		return ports.String()
	}
	for _, port := range p.Ports {
		ports.WriteString(fmt.Sprintf("%s://%s:%s\t%s\n", port.Protocol, port.Ip, port.Port, port.Status))
	}
	return strings.TrimSpace(ports.String())
}
