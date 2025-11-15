package main

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/cmd/cli"
)

func main() {
	err := cli.Start()
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
}
