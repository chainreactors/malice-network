package main

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/cli"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/tui"
)

func init() {
	logs.Log.SetFormatter(tui.DefaultLogStyle)
	console.Log.SetFormatter(tui.DefaultLogStyle)
}

func main() {
	cli.StartConsole()
}
