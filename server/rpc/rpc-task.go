package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
)

func (rpc *Server) GetTasks(ctx context.Context, session *clientpb.Session) (*clientpb.Tasks, error) {
	resp := &clientpb.Tasks{
		Tasks: []*clientpb.Task{},
	}

	//for _, task := range session.Tasks {
	//	resp.Tasks = append(resp.Tasks, task.ToProtobuf())
	//}
	return resp, nil
}
