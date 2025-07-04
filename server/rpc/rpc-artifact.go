package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/build"
	"github.com/chainreactors/malice-network/server/internal/db"
	"os"
)

func (rpc *Server) DownloadArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	artifact, err := db.GetArtifactByName(req.Name)
	if err != nil {
		return nil, err
	}
	var data []byte
	switch req.Format {
	case "srdi", "shellcode", "raw", "bin":
		target, ok := consts.GetBuildTarget(artifact.Target)
		if !ok {
			return nil, errs.ErrInvalidateTarget
		}
		if artifact.Type == consts.CommandBuildPulse {
			data, err = build.OBJCOPYPulse(artifact, target.OS, target.Arch)
			if err != nil {
				return nil, err
			}
		} else {
			if target.OS != consts.Windows {
				return nil, errs.ErrInvalidateTarget
			}
			data, err = build.SRDIArtifact(artifact, target.OS, target.Arch)
			if err != nil {
				return nil, err
			}
		}
	default:
		data, err = os.ReadFile(artifact.Path)
		if err != nil {
			return nil, err
		}
		artifact.Name = build.GetFilePath(artifact.Name, artifact.Target, artifact.Type, req.Format)
	}

	result := artifact.ToArtifact(data)
	return result, nil
}

func (rpc *Server) UploadArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	if req.Name == "" {
		req.Name = codenames.GetCodename()
	}
	artifact, err := db.SaveArtifact(req.Name, req.Type, req.Platform, req.Arch, req.Stage, consts.ArtifactFromUpload)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(artifact.Path, req.Bin, 0644)
	if err != nil {
		return nil, err
	}
	return artifact.ToArtifact([]byte{}), nil
}

// for listener
func (rpc *Server) GetArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	artifact, err := db.GetArtifact(req)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(artifact.Path)
	if err != nil {
		return nil, err
	}
	return artifact.ToArtifact(data), nil
}

func (rpc *Server) ListArtifact(ctx context.Context, req *clientpb.Empty) (*clientpb.Artifacts, error) {
	artifacts, err := db.GetArtifacts()
	if err != nil {
		return nil, err
	}
	return artifacts, nil
}

func (rpc *Server) FindArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	return db.FindArtifact(req)
}

func (rpc *Server) DeleteArtifact(ctx context.Context, req *clientpb.Artifact) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, db.DeleteArtifactByName(req.Name)
}
