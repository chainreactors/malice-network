package main

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/cli"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
)

func init() {
	logs.Log.SetFormatter(utils.DefaultLogStyle)
	console.Log.SetFormatter(utils.DefaultLogStyle)
}

func main() {
	cli.StartConsole()
}
