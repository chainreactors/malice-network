package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func (rpc *Server) NewProfile(ctx context.Context, req *clientpb.Profile) (*clientpb.Empty, error) {
	err := db.NewProfile(req)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) GetProfiles(ctx context.Context, req *clientpb.Empty) (*clientpb.Profiles, error) {
	var profiles clientpb.Profiles
	profiles.Profiles = make([]*clientpb.Profile, 0)
	profilesDB, err := db.GetProfiles()
	if err != nil {
		return nil, err
	}
	for _, profile := range profilesDB {
		profiles.Profiles = append(profiles.Profiles, &clientpb.Profile{
			Name:       profile.Name,
			Target:     profile.Target,
			Type:       profile.Type,
			PipelineId: profile.PipelineID,
		})
	}

	return &profiles, nil
}
