package reg

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

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

	// 获取参数
	valueName, _ := cmd.Flags().GetString("value")
	valueType, _ := cmd.Flags().GetString("type")
	data, _ := cmd.Flags().GetString("data")

	session := con.GetInteractive()
	task, err := RegAdd(con.Rpc, session, hive, path, valueName, valueType, data)
	if err != nil {
		return err
	}

	session.Console(task, fmt.Sprintf("add or modify registry key: %s\\%s\\%s", hive, path, valueName))
	return nil
}

func RegAdd(rpc clientrpc.MaliceRPCClient, session *core.Session, hive, path string, valueName, valueType, data string) (*clientpb.Task, error) {
	request := &implantpb.RegistryWriteRequest{
		Hive: hive,
		Path: fileutils.FormatWindowPath(path),
		Key:  valueName,
	}

	// 根据类型转换数据
	switch valueType {
	case "REG_SZ":
		request.StringValue = data
		request.Regtype = 1
	case "REG_BINARY":
		// 将十六进制字符串转换为字节数组
		byteValue, _ := hex.DecodeString(strings.ReplaceAll(data, " ", ""))
		request.ByteValue = byteValue
		request.Regtype = 3
	case "REG_DWORD":
		// 将字符串转换为 uint32
		if v, err := strconv.ParseUint(data, 0, 32); err == nil {
			request.DwordValue = uint32(v)
		}
		request.Regtype = 4
	case "REG_QWORD":
		// 将字符串转换为 uint64
		if v, err := strconv.ParseUint(data, 0, 64); err == nil {
			request.QwordValue = v
		}
		request.Regtype = 11
	default:
		// 默认使用 REG_SZ
		request.StringValue = data
		request.Regtype = 1
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
		consts.ModuleRegAdd+"(active(),\"HKEY_LOCAL_MACHINE\",\"SOFTWARE\\Example\",\"TestValue\",\"REG_DWORD\",\"1\")",
		[]string{
			"session: special session",
			"hive: registry hive",
			"path: registry path",
			"valueName: value name",
			"valueType: value type (REG_SZ, REG_BINARY, REG_DWORD, REG_QWORD)",
			"data: value data",
		},
		[]string{"task"})
}
