package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func (rpc *Server) NewProfile(ctx context.Context, req *clientpb.Profile) (*clientpb.Empty, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("profile name cannot be empty")
	}

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
		profiles.Profiles = append(profiles.Profiles, profile.ToProtobuf())
	}

	return &profiles, nil
}

func (rpc *Server) DeleteProfile(ctx context.Context, req *clientpb.Profile) (*clientpb.Empty, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("profile name cannot be empty")
	}

	err := db.DeleteProfileByName(req.Name)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) UpdateProfile(ctx context.Context, req *clientpb.Profile) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, db.UpdateProfileRaw(req.Name, req.Content)
}
