package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"os"
	"path/filepath"
)

func (rpc *Server) DownloadArtifact(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	builder, err := db.GetArtifactByName(req.Name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(builder.Path)
	if err != nil {
		return nil, err
	}
	result := builder.ToProtobuf(data)
	result.Name = result.Name + filepath.Ext(builder.Path)
	return result, nil
}

func (rpc *Server) UploadArtifact(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	if req.Name == "" {
		req.Name = codenames.GetCodename()
	}
	builder, err := db.SaveArtifact(req.Name, req.Type, req.Platform, req.Arch, req.Stage)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(builder.Path, req.Bin, 0644)
	if err != nil {
		return nil, err
	}
	return builder.ToProtobuf(nil), nil
}

// for listener
func (rpc *Server) GetArtifact(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	builder, err := db.GetArtifactById(req.Id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(builder.Path)
	if err != nil {
		return nil, err
	}

	return builder.ToProtobuf(data), nil
}

func (rpc *Server) FindArtifact(ctx context.Context, req *clientpb.Builder) (*clientpb.Builder, error) {
	return db.FindArtifact(req)
}
