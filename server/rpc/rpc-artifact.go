package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/db"
	"os"
)

func (rpc *Server) DownloadArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	var path string
	builder, err := db.GetArtifactByName(req.Name)
	if err != nil {
		return nil, err
	}
	if builder.IsSRDI && req.IsSrdi {
		path = builder.ShellcodePath
	} else {
		path = builder.Path
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	builder.Name = build.GetFilePath(builder.Name, builder.Target, builder.Type, builder.IsSRDI)
	result := builder.ToArtifact(data)
	return result, nil
}

func (rpc *Server) UploadArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Builder, error) {
	if req.Name == "" {
		req.Name = codenames.GetCodename()
	}
	builder, err := db.SaveArtifact(req.Name, req.Type, req.Platform, req.Arch, req.Stage, consts.ArtifactFromUpload)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(builder.Path, req.Bin, 0644)
	if err != nil {
		return nil, err
	}
	return builder.ToProtobuf(), nil
}

// for listener
func (rpc *Server) GetArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	builder, err := db.GetArtifactById(req.Id)
	if err != nil {
		return nil, err
	}
	var data []byte
	if builder.ShellcodePath == "" {
		data, err = build.SRDIArtifact(builder, builder.Os, builder.Arch)
	} else {
		data, err = os.ReadFile(builder.ShellcodePath)
	}
	if err != nil {
		return nil, err
	}
	return builder.ToArtifact(data), nil
}

func (rpc *Server) ListBuilder(ctx context.Context, req *clientpb.Empty) (*clientpb.Builders, error) {
	builders, err := db.GetBuilders()
	if err != nil {
		return nil, err
	}
	return builders, nil
}

func (rpc *Server) FindArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	return db.FindArtifact(req)
}

func (rpc *Server) DeleteArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, db.DeleteArtifactByName(req.Name)
}

func (rpc *Server) GetArtifactsByProfile(ctx context.Context, req *clientpb.Profile) (*clientpb.Builders, error) {
	builders, err := db.GetBuilderByProfileName(req.Name)
	if err != nil {
		return nil, err
	}
	return builders, nil
}
