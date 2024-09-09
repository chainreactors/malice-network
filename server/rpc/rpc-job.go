package rpc

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) JobStream(stream listenerrpc.ListenerRPC_JobStreamServer) error {
	go func() {
		for {
			select {
			case msg := <-core.Jobs.Ctrl:
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
			if msg.Ctrl == consts.CtrlPipelineStart {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventPipeline,
					Message:   fmt.Sprintf("%s start", msg.Job.GetPipeline().GetTcp().GetName()),
				})
			} else if msg.Ctrl == consts.CtrlPipelineStop {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventPipeline,
					Message:   fmt.Sprintf("%s stop", msg.Job.GetPipeline().GetTcp().GetName()),
				})
			} else if msg.Ctrl == consts.CtrlWebsiteStart {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventWebsite,
					Message:   fmt.Sprintf("%s start", msg.Job.GetPipeline().GetWeb().GetName()),
				})
			} else if msg.Ctrl == consts.CtrlWebsiteStop {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventWebsite,
					Message:   fmt.Sprintf("%s stop", msg.Job.GetPipeline().GetWeb().GetName()),
				})
			} else if msg.Ctrl == consts.RegisterWebsite {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventWebsite,
					Op:        consts.CtrlWebUpload,
					Message:   fmt.Sprintf("website register"),
				})
			}
		} else {
			if msg.Ctrl == consts.CtrlWebsiteStart || msg.Ctrl == consts.CtrlWebsiteStop {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventWebsite,
					Err:       fmt.Sprintf("%d, %s", msg.Status, msg.Error),
				})
			} else {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventPipeline,
					Err:       fmt.Sprintf("%d, %s", msg.Status, msg.Error),
				})
			}
		}
	}
}
