package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) JobStream(stream listenerrpc.ListenerRPC_JobStreamServer) error {

	go func() {
		for {
			select {
			case msg := <-core.Jobs.Ctrl:
				err := stream.Send(msg)
				if err != nil {
					logs.Log.Errorf("send job ctrl faild %v", err)
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
				IsNotify:  true,
				Job:       msg.Job,
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

func (rpc *Server) ListJobs(ctx context.Context, req *clientpb.Empty) (*clientpb.Pipelines, error) {
	var pipelines []*clientpb.Pipeline
	for _, job := range core.Jobs.All() {
		pipeline, ok := job.Message.(*clientpb.Pipeline)
		if !ok {
			continue
		}
		if pipeline.GetTcp() != nil {
			pipelines = append(pipelines, job.Message.(*clientpb.Pipeline))
		}
	}
	return &clientpb.Pipelines{Pipelines: pipelines}, nil
}
