package listener

import (
	"encoding/json"

	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/spf13/cobra"
)

func NewHttpPipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID, host, port := common.ParsePipelineFlags(cmd)
	target, beaconPipeline := common.ParseArtifactFlags(cmd)
	name := cmd.Flags().Arg(0)
	if port == 0 {
		port = cryptography.RandomInRange(10240, 65535)
	}

	// 解析TLS和加密配置
	tls, err := common.ParseTLSFlags(cmd)
	if err != nil {
		return err
	}
	parser, encryption := common.ParseEncryptionFlags(cmd)
	if parser == "default" {
		parser = consts.ImplantMalefic
	}

	// 解析HTTP特定的参数
	headers, _ := cmd.Flags().GetStringToString("headers")
	errorPage, _ := cmd.Flags().GetString("error-page")
	bodyPrefix, _ := cmd.Flags().GetString("body-prefix")
	bodySuffix, _ := cmd.Flags().GetString("body-suffix")

	// 转换headers格式
	headerMap := make(map[string][]string)
	for k, v := range headers {
		headerMap[k] = []string{v}
	}

	// 创建HTTP特定参数
	params := &types.PipelineParams{
		Headers:    headerMap,
		ErrorPage:  errorPage,
		BodyPrefix: bodyPrefix,
		BodySuffix: bodySuffix,
	}

	// 序列化参数
	paramsJson, err := json.Marshal(params)
	if err != nil {
		return err
	}

	// 注册pipeline
	_, err = con.Rpc.RegisterPipeline(con.Context(), &clientpb.Pipeline{
		Encryption:     encryption,
		Tls:            tls,
		Name:           name,
		ListenerId:     listenerID,
		Target:         target,
		Parser:         parser,
		BeaconPipeline: beaconPipeline,
		Enable:         false,
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Name:   name,
				Host:   host,
				Port:   port,
				Params: string(paramsJson),
			},
		},
	})
	if err != nil {
		return err
	}

	con.Log.Importantf("HTTP Pipeline %s registered\n", name)
	_, err = con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:           name,
		ListenerId:     listenerID,
		Target:         target,
		BeaconPipeline: beaconPipeline,
	})
	if err != nil {
		return err
	}
	return nil
}
