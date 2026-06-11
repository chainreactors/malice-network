package basic

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/spf13/cobra"
)

func SwitchCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	pipeline, _ := cmd.Flags().GetString("pipeline")
	if pipeline == "" {
		return fmt.Errorf("must specify --pipeline")
	}

	pipe, ok := con.Pipelines[pipeline]
	if !ok {
		return fmt.Errorf("no such pipeline: %s", pipeline)
	}

	task, err := Switch(con.Rpc, session, pipe)
	if err != nil {
		return err
	}
	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func Switch(rpc clientrpc.MaliceRPCClient, session *client.Session, pipeline *clientpb.Pipeline) (*clientpb.Task, error) {
	req, err := buildSwitchRequest(pipeline)
	if err != nil {
		return nil, err
	}
	return rpc.Switch(session.Context(), req)
}

func buildSwitchRequest(pipeline *clientpb.Pipeline) (*implantpb.Switch, error) {
	target, err := buildSwitchTarget(pipeline)
	if err != nil {
		return nil, err
	}

	return &implantpb.Switch{
		Targets: []*implantpb.Target{target},
		Action:  implantpb.SwitchAction_REPLACE,
		Key:     switchPipelineKey(pipeline),
	}, nil
}

func buildSwitchTarget(pipeline *clientpb.Pipeline) (*implantpb.Target, error) {
	if pipeline == nil {
		return nil, fmt.Errorf("pipeline is nil")
	}

	address, err := switchPipelineAddress(pipeline)
	if err != nil {
		return nil, err
	}

	target := &implantpb.Target{
		Address: address,
	}

	switch {
	case pipeline.GetTcp() != nil:
		target.Protocol = "tcp"
		proxy, err := buildSwitchProxyConfig(pipeline.GetTcp().GetProxy())
		if err != nil {
			return nil, err
		}
		target.ProxyConfig = proxy
	case pipeline.GetHttp() != nil:
		target.Protocol = "http"
		target.HttpConfig, err = buildSwitchHTTPConfig(pipeline)
		if err != nil {
			return nil, err
		}
		proxy, err := buildSwitchProxyConfig(pipeline.GetHttp().GetProxy())
		if err != nil {
			return nil, err
		}
		target.ProxyConfig = proxy
	case pipeline.GetRem() != nil:
		target.Protocol = "rem"
		target.RemConfig = &implantpb.TargetRemConfig{Link: pipeline.GetRem().GetLink()}
	default:
		return nil, fmt.Errorf("pipeline %s (%s) is not switchable", pipeline.GetName(), pipeline.GetType())
	}

	target.TlsConfig = buildSwitchTLSConfig(pipeline)
	return target, nil
}

func switchPipelineAddress(pipeline *clientpb.Pipeline) (string, error) {
	if pipeline == nil {
		return "", fmt.Errorf("pipeline is nil")
	}

	host := strings.TrimSpace(pipeline.GetIp())
	switch {
	case pipeline.GetTcp() != nil:
		if host == "" {
			host = strings.TrimSpace(pipeline.GetTcp().GetHost())
		}
		port := pipeline.GetTcp().GetPort()
		if host == "" || port == 0 {
			return "", fmt.Errorf("tcp pipeline %s address is incomplete", pipeline.GetName())
		}
		return net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10)), nil
	case pipeline.GetHttp() != nil:
		if host == "" {
			host = strings.TrimSpace(pipeline.GetHttp().GetHost())
		}
		port := pipeline.GetHttp().GetPort()
		if host == "" || port == 0 {
			return "", fmt.Errorf("http pipeline %s address is incomplete", pipeline.GetName())
		}
		return net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10)), nil
	case pipeline.GetRem() != nil:
		if host == "" {
			host = strings.TrimSpace(pipeline.GetRem().GetHost())
		}
		port := pipeline.GetRem().GetPort()
		if host == "" || port == 0 {
			return "", fmt.Errorf("rem pipeline %s address is incomplete", pipeline.GetName())
		}
		return net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10)), nil
	default:
		return "", fmt.Errorf("pipeline %s (%s) is not switchable", pipeline.GetName(), pipeline.GetType())
	}
}

func buildSwitchTLSConfig(pipeline *clientpb.Pipeline) *implantpb.TargetTlsConfig {
	tls := pipeline.GetTls()
	if tls == nil || !tls.GetEnable() {
		return nil
	}

	return &implantpb.TargetTlsConfig{
		Enable:     true,
		Sni:        strings.TrimSpace(tls.GetDomain()),
		SkipVerify: true,
	}
}

func buildSwitchProxyConfig(raw string) (*implantpb.TargetProxyConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse proxy %q: %w", raw, err)
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme == "" {
		host, port, splitErr := net.SplitHostPort(raw)
		if splitErr != nil {
			return nil, fmt.Errorf("parse proxy %q: %w", raw, splitErr)
		}
		portUint, convErr := strconv.ParseUint(port, 10, 32)
		if convErr != nil {
			return nil, fmt.Errorf("parse proxy port %q: %w", port, convErr)
		}
		return &implantpb.TargetProxyConfig{
			Type: "http",
			Host: host,
			Port: uint32(portUint),
		}, nil
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("proxy %q host is empty", raw)
	}

	portStr := parsed.Port()
	if portStr == "" {
		return nil, fmt.Errorf("proxy %q port is empty", raw)
	}

	portUint, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parse proxy port %q: %w", portStr, err)
	}

	proxy := &implantpb.TargetProxyConfig{
		Type: scheme,
		Host: host,
		Port: uint32(portUint),
	}
	if parsed.User != nil {
		proxy.Username = parsed.User.Username()
		proxy.Password, _ = parsed.User.Password()
	}
	return proxy, nil
}

func buildSwitchHTTPConfig(pipeline *clientpb.Pipeline) (*implantpb.TargetHttpConfig, error) {
	http := pipeline.GetHttp()
	if http == nil {
		return nil, nil
	}

	params, err := implanttypes.UnmarshalPipelineParams(http.GetParams())
	if err != nil {
		return nil, fmt.Errorf("unmarshal http pipeline params for %s: %w", pipeline.GetName(), err)
	}

	headers := make(map[string]string, len(params.Headers))
	for key, values := range params.Headers {
		if len(values) == 0 {
			continue
		}
		headers[key] = values[0]
	}

	return &implantpb.TargetHttpConfig{
		Method:  "POST",
		Path:    "/",
		Version: "1.1",
		Headers: headers,
	}, nil
}

func switchPipelineKey(pipeline *clientpb.Pipeline) []byte {
	if pipeline == nil {
		return nil
	}

	for _, encryption := range pipeline.GetEncryption() {
		if encryption == nil {
			continue
		}
		key := strings.TrimSpace(encryption.GetKey())
		if key == "" {
			continue
		}
		if strings.EqualFold(encryption.GetType(), consts.CryptorRAW) {
			return nil
		}
		return []byte(key)
	}
	return nil
}
