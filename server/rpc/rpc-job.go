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
			if msg.Ctrl == consts.CtrlWebUpload {
				continue
			}
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventJob,
				Op:        msg.Ctrl,
				Message:   fmt.Sprintf("%s", msg.Job.Name),
			})
		} else {
			if msg.Ctrl == consts.CtrlWebUpload {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventWebsite,
					Op:        msg.Ctrl,
					Err:       fmt.Sprintf("status %d,  %s", msg.Status, msg.Error),
				})
				continue
			}
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventJob,
				Op:        msg.Ctrl,
				Err:       fmt.Sprintf("%s faild,status %d,  %s", msg.Job.Name, msg.Status, msg.Error),
			})
		}
	}
}
