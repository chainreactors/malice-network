package output

import (
	"bytes"
	"encoding/json"
	"github.com/chainreactors/parsers"
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
