package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/protobuf/proto"
)

func (rpc *Server) Build(ctx context.Context, req *clientpb.BuildConfig) (*clientpb.Artifact, error) {
	if req.Type == consts.CommandBuildPulse {
		var artifactID uint32
		if req.ArtifactId != 0 {
			artifactID = req.ArtifactId
		} else {
			profile, _ := db.GetProfile(req.ProfileName)
			yamlID := profile.Pulse.Flags.ArtifactID
			if uint32(yamlID) != 0 {
				artifactID = yamlID
			} else {
				artifactID = 0
			}
		}
		builders, err := db.FindBeaconArtifact(artifactID, req.ProfileName)
		if err != nil {
			return nil, err
		}
		if len(builders) > 0 {
			artifactID = builders[0].ID
			req.ArtifactId = artifactID
		} else {
			beaconReq := proto.Clone(req).(*clientpb.BuildConfig)
			if req.Source == consts.ArtifactFromAction {
				beaconReq.Inputs["package"] = consts.CommandBuildBeacon
				if beaconReq.Inputs["targets"] == consts.TargetX86Windows {
					beaconReq.Inputs["targets"] = consts.TargetX86WindowsGnu
				} else {
					beaconReq.Inputs["targets"] = consts.TargetX64WindowsGnu
				}
			} else {
				beaconReq.Type = consts.CommandBuildBeacon
				if beaconReq.Target == consts.TargetX86Windows {
					beaconReq.Target = consts.TargetX86WindowsGnu
				} else {
					beaconReq.Target = consts.TargetX64WindowsGnu
				}
			}
			beaconBuilder := build.NewBuilder(beaconReq)
			artifact, err := beaconBuilder.GenerateConfig()
			if err != nil {
				return nil, err
			}
			req.ArtifactId = artifact.Id
			go func() {
				executeErr := beaconBuilder.ExecuteBuild()
				if executeErr == nil {
					beaconBuilder.CollectArtifact()
				} else {
					logs.Log.Errorf("failed to build %s: %s", artifact.Name, executeErr)
					build.SendFailedMsg(artifact)
				}
			}()
		}
	}

	builder := build.NewBuilder(req)
	artifact, err := builder.GenerateConfig()
	if err != nil {
		return nil, err
	}
	go func() {
		executeErr := builder.ExecuteBuild()
		if executeErr == nil {
			builder.CollectArtifact()
		} else {
			logs.Log.Errorf("failed to build %s: %s", artifact.Name, executeErr)
		}
	}()

	return artifact, nil
}

func (rpc *Server) BuildLog(ctx context.Context, req *clientpb.Artifact) (*clientpb.Artifact, error) {
	resultLog, err := db.GetBuilderLogs(req.Id, int(req.LogNum))
	if err != nil {
		return nil, err
	}
	req.Log = []byte(resultLog)
	return req, nil
}

func (rpc *Server) DockerStatus(ctx context.Context, req *clientpb.Empty) (*clientpb.Empty, error) {
	cli, err := build.GetDockerClient()
	if err != nil {
		return nil, err
	}
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) WorkflowStatus(ctx context.Context, req *clientpb.GithubWorkflowConfig) (*clientpb.Empty, error) {
	if req.Owner == "" || req.Repo == "" || req.Token == "" {
		config := configs.GetGithubConfig()
		if config == nil {
			return nil, fmt.Errorf("please set github config use flag or server config")
		}
		req = config.ToProtobuf()
	}
	err := build.GetWorkflowStatus(req)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}
