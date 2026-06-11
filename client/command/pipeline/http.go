package pipeline

import (
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/spf13/cobra"
	"os"
)

func NewHttpPipelineCmd(cmd *cobra.Command, con *core.Console) error {
	listenerID, proxy, host, port := common.ParsePipelineFlags(cmd)
	if port == 0 {
		port = cryptography.RandomInRange(10240, 65535)
	}
	name := cmd.Flags().Arg(0)
	if name == "" {
		name = fmt.Sprintf("http_%s_%d", listenerID, port)
	}

	// 解析TLS和加密配置
	tls, certName, err := common.ParseTLSFlags(cmd)
	if err != nil {
		return err
	}
	parser, encryption := common.ParseEncryptionFlags(cmd)
	if parser == "default" {
		parser = consts.ImplantMalefic
	}

	secure := common.ParseSecureFlags(cmd)

	// 解析HTTP特定的参数
	headers, _ := cmd.Flags().GetStringToString("headers")
	errorPage, _ := cmd.Flags().GetString("error-page")
	bodyPrefix, _ := cmd.Flags().GetString("body-prefix")
	bodySuffix, _ := cmd.Flags().GetString("body-suffix")
	if errorPage != "" {
		content, err := os.ReadFile(errorPage)
		if err != nil {
			return err
		}
		errorPage = string(content)
	}

	// 转换headers格式
	headerMap := make(map[string][]string)
	for k, v := range headers {
		headerMap[k] = []string{v}
	}

	// 创建HTTP特定参数
	params := &implanttypes.PipelineParams{
		Headers:    headerMap,
		ErrorPage:  errorPage,
		BodyPrefix: bodyPrefix,
		BodySuffix: bodySuffix,
	}
	pipeline := &clientpb.Pipeline{
		Encryption: encryption,
		Tls:        tls,
		Secure:     secure,
		Name:       name,
		ListenerId: listenerID,
		Parser:     parser,
		CertName:   certName,
		Enable:     false,
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Name:   name,
				Host:   host,
				Port:   port,
				Params: params.String(),
				Proxy:  proxy,
			},
		},
	}
	// 注册pipeline
	_, err = con.Rpc.RegisterPipeline(con.Context(), pipeline)
	if err != nil {
		return err
	}

	con.Log.Importantf("HTTP Pipeline %s registered\n", name)
	_, err = con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
		Pipeline:   pipeline,
	})
	if err != nil {
		return err
	}
	return nil
}
