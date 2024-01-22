package main

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/cli"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/styles"
)

func init() {
	styles.DefaultLogFormatter(logs.Log)
	styles.DefaultLogFormatter(console.Log)
}

func main() {
	cli.StartConsole()
}
