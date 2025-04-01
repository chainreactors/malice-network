package third

import (
	"strings"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

func CurlCmd(cmd *cobra.Command, con *repl.Console) error {
	url := cmd.Flags().Arg(0)
	method, _ := cmd.Flags().GetString("method")
	timeout, _ := cmd.Flags().GetInt("timeout")
	body, _ := cmd.Flags().GetString("body")
	headers, _ := cmd.Flags().GetStringArray("header")

	headerMap := make(map[string]string)
	for _, h := range headers {
		if parts := strings.SplitN(h, ":", 2); len(parts) == 2 {
			headerMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	session := con.GetInteractive()
	task, err := Curl(con.Rpc, session, url, method, int32(timeout), []byte(body), headerMap)
	if err != nil {
		return err
	}

	session.Console(task, "curl "+url)
	return nil
}

func Curl(rpc clientrpc.MaliceRPCClient, sess *core.Session, url string, method string, timeout int32, body []byte, headers map[string]string) (*clientpb.Task, error) {
	task, err := rpc.Curl(sess.Context(), &implantpb.CurlRequest{
		Url:     url,
		Method:  method,
		Timeout: timeout,
		Body:    body,
		Header:  headers,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterCurlFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		"curl",
		Curl,
		"bcurl",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, url string) (*clientpb.Task, error) {
			return Curl(rpc, sess, url, "GET", 30, nil, nil)
		},
		output.ParseBinaryResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		"curl",
		"curl",
		`curl(active(),"http://example.com","GET",30,nil,nil)`,
		[]string{
			"session: special session",
			"url: target url",
			"method: HTTP method",
			"timeout: request timeout in seconds",
			"body: request body",
			"headers: request headers",
		},
		[]string{"task"})

	con.AddCommandFuncHelper(
		"bcurl",
		"bcurl",
		`bcurl(active(),"http://example.com")`,
		[]string{
			"session: special session",
			"url: target url",
		},
		[]string{"task"})
}
