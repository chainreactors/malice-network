package rpc

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
	"gopkg.in/yaml.v3"
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
			var nMsg string
			switch msg.Job.Pipeline.Body.(type) {
			case *lispb.Pipeline_Tcp:
				marshal, err := yaml.Marshal(msg.Job.Pipeline.GetTcp())
				if err != nil {
					return err
				}
				nMsg = string(marshal)
			case *lispb.Pipeline_Web:
				getWeb := msg.Job.Pipeline.GetWeb()
				getWeb.Contents = nil
				marshal, err := yaml.Marshal(getWeb)
				if err != nil {
					return err
				}
				nMsg = string(marshal)
			}
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventJob,
				Op:        msg.Ctrl,
				Message:   nMsg,
				IsNotify:  true,
			})
			if err != nil {
				return err
			}
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
