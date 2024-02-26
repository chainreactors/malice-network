package rpc

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	core2 "github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) JobStream(stream listenerrpc.ListenerRPC_JobStreamServer) error {
	go func() {
		for {
			select {
			case msg := <-core2.Jobs.Ctrl:
				err := stream.Send(msg)
				if err != nil {
					return
				}
			}
		}
	}()

	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		if msg.Status == consts.CtrlStatusSuccess {
			core2.EventBroker.Publish(core2.Event{
				Job:       core2.Jobs.Get(msg.Job.Id),
				EventType: consts.EventPipelineStart,
			})
		} else {
			core2.EventBroker.Publish(core2.Event{
				EventType: consts.EventPipelineError,
				Err:       fmt.Sprintf("%d, %s", msg.Status, msg.Error),
			})
		}
		// todo stop pipeline
	}
}
