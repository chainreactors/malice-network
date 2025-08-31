package clientpb

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
)

func (pipe *Pipeline) Address() string {
	switch body := pipe.Body.(type) {
	case *Pipeline_Http:
		return fmt.Sprintf("%s:%d", pipe.Ip, body.Http.Port)
	case *Pipeline_Tcp:
		return fmt.Sprintf("%s:%d", pipe.Ip, body.Tcp.Port)
	case *Pipeline_Rem:
		return fmt.Sprintf("%s:%d", pipe.Ip, body.Rem.Port)
	default:
		return ""
	}
}

func (task *Task) Progress() string {
	if task.Total == -1 {
		return fmt.Sprintf("%d/∞", task.Cur)
	} else {
		return fmt.Sprintf("%d/%d", task.Cur, task.Total)
	}
}

func (pipe *Pipeline) URL() string {
	scheme := "http"
	if pipe.Tls.Enable {
		scheme = "https"
	}

	if pipe.Type == consts.WebsitePipeline {
		web := pipe.GetWeb()
		// baseURL 只到 host:port
		return fmt.Sprintf("%s://%s:%d", scheme, pipe.Ip, web.Port) + web.Root
	} else if pipe.Type == consts.HTTPPipeline {
		return fmt.Sprintf("%s://%s:%d", scheme, pipe.Ip, pipe.GetHttp().Port)
	} else if pipe.Type == consts.TCPPipeline {
		return fmt.Sprintf("tcp://%s:%d", pipe.Ip, pipe.GetTcp().Port)
	}

	return ""
}

func (pipe *Job) FirstContent() *WebContent {
	for _, content := range pipe.Contents {
		return content
	}
	return nil
}

func (pipe *Pipeline) KVMap() (map[string]interface{}, []string) {
	pipelineMap := map[string]interface{}{
		"Name":        pipe.Name,
		"Type":        pipe.Type,
		"Listener ID": pipe.ListenerId,
	}

	var orderedKeys []string
	orderedKeys = append(orderedKeys, "Name", "Type", "Listener ID")

	switch pipe.Body.(type) {
	case *Pipeline_Tcp:
		pipelineMap["Address"] = pipe.Address()
		pipelineMap["TLS"] = pipe.Tls.Enable
		pipelineMap["Cert"] = pipe.CertName
		orderedKeys = append(orderedKeys, "Address", "TLS", "Cert")
	case *Pipeline_Http:
		pipelineMap["Address"] = pipe.Address()
		pipelineMap["TLS"] = pipe.Tls.Enable
		pipelineMap["Cert"] = pipe.CertName
		orderedKeys = append(orderedKeys, "Address", "TLS", "Cert")
	case *Pipeline_Bind:
		pipelineMap["Ip"] = pipe.Ip
		orderedKeys = append(orderedKeys, "Ip")
	case *Pipeline_Rem:
		pipelineMap["Address"] = pipe.Address()
		orderedKeys = append(orderedKeys, "Address")
	case *Pipeline_Web:
		pipelineMap["Port"] = pipe.GetWeb().Port
		pipelineMap["URL"] = pipe.URL()
		pipelineMap["TLS"] = pipe.Tls.Enable
		pipelineMap["Cert"] = pipe.CertName
		orderedKeys = append(orderedKeys, "Port", "URL", "TLS", "Cert")
	}

	return pipelineMap, orderedKeys
}
