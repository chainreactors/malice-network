package main

import "github.com/chainreactors/malice-network/server/cmd/server"

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

	server.Execute()
}
