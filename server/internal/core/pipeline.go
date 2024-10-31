package core

import "github.com/chainreactors/malice-network/helper/proto/client/clientpb"

type Pipeline interface {
	ID() string
	Start() error
	Close() error
	ToProtobuf() *clientpb.Pipeline
}

type Pipelines map[string]Pipeline

func (ps Pipelines) Add(p Pipeline) {
	ps[p.ID()] = p
}

func (ps Pipelines) Get(id string) Pipeline {
	return ps[id]
}

func (ps Pipelines) ToProtobuf() *clientpb.Pipelines {
	var pls = &clientpb.Pipelines{
		Pipelines: make([]*clientpb.Pipeline, 0),
	}
	for _, p := range ps {
		pls.Pipelines = append(pls.Pipelines, p.ToProtobuf())
	}
	return pls
}
