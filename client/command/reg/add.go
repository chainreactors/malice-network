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
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/spf13/cobra"
)

// RegAddCmd adds or modifies a registry key value.
func RegAddCmd(cmd *cobra.Command, con *repl.Console) error {
	// 解析注册表的各项参数
	path := cmd.Flags().Arg(0)
	hive, path := FormatRegPath(path)
	key := cmd.Flags().Arg(1)
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
		Path:        fileutils.FormatWindowPath(path),
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
	con.AddCommandFuncHelper(
		consts.ModuleRegAdd,
		consts.ModuleRegAdd,
		consts.ModuleRegAdd+"(active(),\"HKEY_LOCAL_MACHINE\",\"SOFTWARE\\Example\",\"TestKey\",\"example\",\"\",1,0,0)",
		[]string{
			"session: special session",
			"hive: registry hive",
			"path: registry path",
			"key: registry",
			"stringValue: string value",
			"byteValue: byte value",
			"dwordValue: dword value",
			"qwordValue: qword value",
			"regtype: registry type",
		},
		[]string{"task"})
}
