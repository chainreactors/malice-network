package main

import (
	_ "embed"
	"github.com/chainreactors/logs"

	"github.com/chainreactors/malice-network/server/cmd/server"
	//_ "net/http/pprof"
)

//go:embed config.yaml
var serverConfig []byte

func main() {
	//f, err := os.Create("cpu.prof")
	//if err != nil {
	//	logs.Log.Errorf("could not create CPU profile: ", err)
	//}
	//defer f.Close()
	//
	//if err := pprof.StartCPUProfile(f); err != nil {
	//	logs.Log.Errorf("could not start CPU profile: ", err)
	//}
	//
	//go func() {
	//	http.ListenAndServe("localhost:6060", nil)
	//}()

	err := server.Start(serverConfig)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
}
