package sys

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
	"strings"
)

// WmiQueryCmd performs a WMI query.
func WmiQueryCmd(cmd *cobra.Command, con *core.Console) error {
	namespace, _ := cmd.Flags().GetString("namespace")
	args, _ := cmd.Flags().GetStringSlice("args")

	session := con.GetInteractive()
	task, err := WmiQuery(con.Rpc, session, namespace, args)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func WmiQuery(rpc clientrpc.MaliceRPCClient, session *client.Session, namespace string, args []string) (*clientpb.Task, error) {
	request := &implantpb.WmiQueryRequest{
		Namespace: namespace,
		Args:      args,
	}
	return rpc.WmiQuery(session.Context(), request)
}

// WmiExecuteCmd executes a WMI method.
func WmiExecuteCmd(cmd *cobra.Command, con *core.Console) error {
	namespace, _ := cmd.Flags().GetString("namespace")
	className, _ := cmd.Flags().GetString("class_name")
	methodName, _ := cmd.Flags().GetString("method_name")
	param_str, _ := cmd.Flags().GetStringSlice("params")
	params := make(map[string]string)
	for _, i := range param_str {
		kv := strings.SplitN(i, "=", 2)
		if len(kv) != 2 || kv[0] == "" {
			return fmt.Errorf("invalid --params value %q: want key=value", i)
		}
		params[kv[0]] = kv[1]
	}
	session := con.GetInteractive()
	task, err := WmiExecute(con.Rpc, session, namespace, className, methodName, params)
	if err != nil {
		return err
	}
	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func WmiExecute(rpc clientrpc.MaliceRPCClient, session *client.Session, namespace, className, methodName string, params map[string]string) (*clientpb.Task, error) {
	request := &implantpb.WmiMethodRequest{
		Namespace:  namespace,
		ClassName:  className,
		MethodName: methodName,
		Params:     params,
	}
	return rpc.WmiExecute(session.Context(), request)
}

func RegisterWmiFunc(con *core.Console) {
	con.RegisterImplantFunc(
		consts.ModuleWmiQuery,
		WmiQuery,
		"",
		nil,
		output.ParseKVResponse,
		output.FormatKVResponse,
	)

	con.AddCommandFuncHelper(
		consts.ModuleWmiQuery,
		consts.ModuleWmiQuery,
		`wmi_query(active(), "root\\cimv2", {"SELECT * FROM Win32_OperatingSystem"})`,
		[]string{
			"sess: special session",
			"namespace: WMI namespace",
			"args: WMI query arguments",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleWmiExec,
		WmiExecute,
		"",
		nil,
		output.ParseKVResponse,
		output.FormatKVResponse,
	)

	con.AddCommandFuncHelper(
		consts.ModuleWmiExec,
		consts.ModuleWmiExec,
		//session *core.Session, namespace string, args []string
		// params map[string]string
		`wmi_execute(active(), "root\\cimv2", "Win32_Process", "Create", {"CommandLine":"cmd.exe"})`,
		[]string{
			"session: special session",
			"namespace: WMI namespace",
			"className: WMI class name",
			"methodName: WMI method name",
			"params: WMI method parameters",
		},
		[]string{"task"})

}
