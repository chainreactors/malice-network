package common

import (
	"fmt"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/client/core"
)

func FindCachedPipeline(con *core.Console, name string, accept func(*clientpb.Pipeline) bool) (*clientpb.Pipeline, error) {
	if con == nil || con.Pipelines == nil {
		return nil, types.ErrNotFoundPipeline
	}
	if accept == nil {
		accept = func(*clientpb.Pipeline) bool { return true }
	}

	if pipeline, ok := con.Pipelines[name]; ok && pipeline != nil {
		if accept(pipeline) {
			return pipeline, nil
		}
	}

	var match *clientpb.Pipeline
	for key, pipeline := range con.Pipelines {
		if key == name || pipeline == nil || pipeline.Name != name || !accept(pipeline) {
			continue
		}
		if match != nil && match != pipeline {
			return nil, fmt.Errorf("pipeline %q is ambiguous; use listener:pipeline", name)
		}
		match = pipeline
	}
	if match != nil {
		return match, nil
	}
	return nil, types.ErrNotFoundPipeline
}
