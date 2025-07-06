package main

import (
	"fmt"
	_ "github.com/chainreactors/malice-network/client/cmd/cli"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/mals"
	"github.com/spf13/cobra"
)

func main() {
	con, err := repl.NewConsole()
	if err != nil {
		fmt.Println(err)
		return
	}
	var cmd = &cobra.Command{
		Use:   "client",
		Short: "",
		Long:  ``,
	}
	cmd.TraverseChildren = true
	command.BindBuiltinCommands(con, cmd)
	command.BindClientsCommands(con)
	rpc := clientrpc.NewMaliceRPCClient(nil)
	intermediate.RegisterBuiltin(rpc)
	command.RegisterClientFunc(con)
	command.RegisterImplantFunc(con)
	vm := plugin.NewLuaVM()
	plug := &plugin.LuaPlugin{DefaultPlugin: &plugin.DefaultPlugin{MalManiFest: &plugin.MalManiFest{}}}
	plug.InitLuaContext(vm)
	mals.GenerateLuaDefinitionFile(vm, intermediate.BuiltinPackage, plugin.ProtoPackage, intermediate.InternalFunctions.Package(intermediate.BuiltinPackage))
	mals.GenerateLuaDefinitionFile(vm, intermediate.RpcPackage, plugin.ProtoPackage, intermediate.InternalFunctions.Package(intermediate.RpcPackage))
	mals.GenerateLuaDefinitionFile(vm, intermediate.BeaconPackage, plugin.ProtoPackage, intermediate.InternalFunctions.Package(intermediate.BeaconPackage))

	mals.GenerateMarkdownDefinitionFile(vm, intermediate.BuiltinPackage, "builtin.md", intermediate.InternalFunctions.Package(intermediate.BuiltinPackage))
	mals.GenerateMarkdownDefinitionFile(vm, intermediate.RpcPackage, "rpc.md", intermediate.InternalFunctions.Package(intermediate.RpcPackage))
	mals.GenerateMarkdownDefinitionFile(vm, intermediate.BeaconPackage, "beacon.md", intermediate.InternalFunctions.Package(intermediate.BeaconPackage))
}
