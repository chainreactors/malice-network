package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"math/rand"
	"time"
)

func newTcpPipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name, host, portUint, certPath, keyPath, tlsEnable := common.ParsePipelineFlags(cmd)
	listenerID := cmd.Flags().Arg(0)
	var cert, key string
	var err error
	if portUint == 0 {
		rand.Seed(time.Now().UnixNano())
		portUint = uint(10000 + rand.Int31n(5001))
	}
	port := uint32(portUint)
	if host == "" {
		host = "0.0.0.0"
	}
	if name == "" {
		name = fmt.Sprintf("%s-tcp-%d", listenerID, port)
	}
	if certPath != "" && keyPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			con.Log.Error(err.Error())
			return
		}
		key, err = cryptography.ProcessPEM(keyPath)
		tlsEnable = true
		if err != nil {
			con.Log.Error(err.Error())
			return
		}
	}
	_, err = con.LisRpc.RegisterPipeline(context.Background(), &clientpb.Pipeline{
		Encryption: &clientpb.Encryption{
			Enable: false,
			Type:   "",
			Key:    "",
		},
		Tls: &clientpb.TLS{
			Cert:   cert,
			Key:    key,
			Enable: tlsEnable,
		},
		Name:       name,
		ListenerId: listenerID,
		Enable:     false,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Host: host,
				Port: port,
			},
		},
	})
	if err != nil {
		con.Log.Error(err.Error())
		return
	}

	_, err = con.LisRpc.StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		con.Log.Error(err.Error())
		return
	}
	con.Log.Importantf("TCP Pipeline %s added\n", name)
}
