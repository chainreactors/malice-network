package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func main() {
	authFile := flag.String("auth", "", "path to admin.auth file")
	action := flag.String("action", "start", "action: list, register, start, stop")
	listenerID := flag.String("listener", "webshell-listener", "listener ID")
	pipelineName := flag.String("pipeline", "webshell_webshell-listener", "pipeline name")
	pipelineType := flag.String("type", "webshell", "pipeline type")
	flag.Parse()

	if *authFile == "" {
		log.Fatal("--auth is required")
	}

	config, err := mtls.ReadConfig(*authFile)
	if err != nil {
		log.Fatalf("read config: %v", err)
	}

	conn, err := mtls.Connect(config)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	server, err := client.NewServerStatus(conn, config)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}

	switch *action {
	case "list":
		listeners, err := server.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
		if err != nil {
			log.Fatalf("get listeners: %v", err)
		}
		for _, l := range listeners.Listeners {
			fmt.Printf("Listener: %s  IP: %s  Active: %v\n", l.Id, l.Ip, l.Active)
			if l.Pipelines != nil {
				for _, p := range l.Pipelines.Pipelines {
					fmt.Printf("  Pipeline: %s  Enable: %v\n", p.Name, p.Enable)
				}
			}
		}

	case "register":
		fmt.Printf("Registering pipeline %s (type=%s) on listener %s\n", *pipelineName, *pipelineType, *listenerID)
		_, err := server.Rpc.RegisterPipeline(context.Background(), &clientpb.Pipeline{
			Name:       *pipelineName,
			ListenerId: *listenerID,
			Type:       *pipelineType,
			Enable:     true,
			Body: &clientpb.Pipeline_Tcp{
				Tcp: &clientpb.TCPPipeline{
					Host: "127.0.0.1",
					Port: 0,
				},
			},
		})
		if err != nil {
			log.Fatalf("register pipeline: %v", err)
		}
		fmt.Println("Pipeline registered!")

	case "start":
		fmt.Printf("Starting pipeline %s on listener %s\n", *pipelineName, *listenerID)
		_, err := server.Rpc.StartPipeline(context.Background(), &clientpb.CtrlPipeline{
			Name:       *pipelineName,
			ListenerId: *listenerID,
		})
		if err != nil {
			log.Fatalf("start pipeline: %v", err)
		}
		fmt.Println("Pipeline started!")

	case "stop":
		fmt.Printf("Stopping pipeline %s on listener %s\n", *pipelineName, *listenerID)
		_, err := server.Rpc.StopPipeline(context.Background(), &clientpb.CtrlPipeline{
			Name:       *pipelineName,
			ListenerId: *listenerID,
		})
		if err != nil {
			log.Fatalf("stop pipeline: %v", err)
		}
		fmt.Println("Pipeline stopped!")
	}
}
