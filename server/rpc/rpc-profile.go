package rpc

import (
	"context"
	"fmt"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func publishProfileLifecycleEvent(operation, name string) {
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventProfile,
		Op:        operation,
		Important: true,
		Message:   fmt.Sprintf("profile %s changed", name),
	})
}

func (rpc *Server) NewProfile(ctx context.Context, req *clientpb.Profile) (*clientpb.Empty, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("profile name cannot be empty")
	}

	err := db.NewProfile(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}
	publishProfileLifecycleEvent(consts.CtrlProfileCreate, req.Name)

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

func (rpc *Server) GetProfileByName(ctx context.Context, req *clientpb.Profile) (*clientpb.Profile, error) {
	return db.GetProfileByNameWithConfig(req.Name)
}

func (rpc *Server) DeleteProfile(ctx context.Context, req *clientpb.Profile) (*clientpb.Empty, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("profile name cannot be empty")
	}

	err := db.DeleteProfileByName(req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete profile: %w", err)
	}
	publishProfileLifecycleEvent(consts.CtrlProfileDelete, req.Name)
	return &clientpb.Empty{}, nil
}

func (rpc *Server) UpdateProfile(ctx context.Context, req *clientpb.Profile) (*clientpb.Empty, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("profile name cannot be empty")
	}
	if _, err := implanttypes.LoadProfile(req.ImplantConfig); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}
	if err := db.UpdateProfileDisk(req.Name, req.ImplantConfig, req.PreludeConfig, req.Resources); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}
	publishProfileLifecycleEvent(consts.CtrlProfileUpdate, req.Name)
	return &clientpb.Empty{}, nil
}
