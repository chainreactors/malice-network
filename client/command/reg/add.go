package reg

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
)

// RegAddCmd adds or modifies a registry key value.
func RegAddCmd(cmd *cobra.Command, con *repl.Console) error {
	// 解析注册表的各项参数
	hive, _ := cmd.Flags().GetString("hive")
	path, _ := cmd.Flags().GetString("path")
	key, _ := cmd.Flags().GetString("key")
	stringValue, _ := cmd.Flags().GetString("string_value")
	byteValue, _ := cmd.Flags().GetBytesBase64("byte_value")
	dwordValue, _ := cmd.Flags().GetUint32("dword_value")
	qwordValue, _ := cmd.Flags().GetUint64("qword_value")
	regtype, _ := cmd.Flags().GetUint32("regtype")

	session := con.GetInteractive()
	task, err := RegAdd(con.Rpc, session, hive, path, key, stringValue, byteValue, dwordValue, qwordValue, regtype)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("add or modify registry key: %s\\%s\\%s", hive, path, key))
	return nil
}

func RegAdd(rpc clientrpc.MaliceRPCClient, session *core.Session, hive, path, key, stringValue string, byteValue []byte, dwordValue uint32, qwordValue uint64, regtype uint32) (*clientpb.Task, error) {
	request := &implantpb.RegistryWriteRequest{
		Hive:        hive,
		Path:        path,
		Key:         key,
		StringValue: stringValue,
		ByteValue:   byteValue,
		DwordValue:  dwordValue,
		QwordValue:  qwordValue,
		Regtype:     regtype,
	}
	return rpc.RegAdd(session.Context(), request)
}

func RegisterRegAddFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleRegAdd,
		RegAdd,
		"",
		nil,
		common.ParseStatus,
		nil,
	)
}
