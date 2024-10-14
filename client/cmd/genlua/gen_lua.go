package main

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/client/repl"
)

func main() {
	con, err := repl.NewConsole()
	if err != nil {
		fmt.Println(err)
		return
	}
	command.RegisterClientFunc(con)
	command.RegisterImplantFunc(con)
	vm := plugin.NewLuaVM()
	plugin.GenerateLuaDefinitionFile(vm, "define.lua")
}
