package common

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/client/core"
)

func FindCachedPipeline(con *core.Console, name string, accept func(*clientpb.Pipeline) bool) (*clientpb.Pipeline, error) {
	if con == nil || con.Server == nil || con.ServerState == nil {
		return nil, types.ErrNotFoundPipeline
	}
	return con.ServerState.FindCachedPipeline(name, accept)
}

func SnapshotCachedPipelines(con *core.Console) map[string]*clientpb.Pipeline {
	if con == nil || con.Server == nil || con.ServerState == nil {
		return nil
	}
	return con.ServerState.SnapshotPipelines()
}
