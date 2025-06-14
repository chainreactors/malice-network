package clientpb

import "fmt"

func (pipe *Pipeline) Address() string {
	switch body := pipe.Body.(type) {
	case *Pipeline_Http:
		return fmt.Sprintf("%s:%d", pipe.Ip, body.Http.Port)
	case *Pipeline_Tcp:
		return fmt.Sprintf("%s:%d", pipe.Ip, body.Tcp.Port)
	default:
		return ""
	}
}
