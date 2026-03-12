package listener

import "github.com/chainreactors/IoM-go/proto/client/clientpb"

// CustomPipeline is a pass-through pipeline for externally managed endpoints.
// The listener does not start any local listener; lifecycle is managed by
// the external process (e.g. CLIProxyAPI bridge).
type CustomPipeline struct {
	name     string
	pipeline *clientpb.Pipeline
}

// NewCustomPipeline creates a new pass-through custom pipeline.
func NewCustomPipeline(pb *clientpb.Pipeline) *CustomPipeline {
	return &CustomPipeline{name: pb.Name, pipeline: pb}
}

func (p *CustomPipeline) ID() string                     { return p.name }
func (p *CustomPipeline) Start() error                   { return nil }
func (p *CustomPipeline) Close() error                   { return nil }
func (p *CustomPipeline) ToProtobuf() *clientpb.Pipeline { return p.pipeline }
