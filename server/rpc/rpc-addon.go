package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func (rpc *Server) ListAddon(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandleWithSession(ctx, req, consts.ModuleListAddon, types.MsgListAddon, func(greq *GenericRequest, spite *implantpb.Spite) {
		if exts := spite.GetAddons(); exts != nil {
			greq.Session.Addons = exts.Addons
			greq.Session.SaveAndNotify("")
		}
	})
}

func (rpc *Server) LoadAddon(ctx context.Context, req *implantpb.LoadAddon) (*clientpb.Task, error) {
	return rpc.GenericInternalWithSession(ctx, req, types.MsgEmpty, func(greq *GenericRequest, spite *implantpb.Spite) {
		greq.Session.Addons = append(greq.Session.Addons, &implantpb.Addon{
			Name:   req.Name,
			Depend: req.Depend,
			Type:   req.Type,
		})
		greq.Session.SaveAndNotify("")
	})
}

func (rpc *Server) ExecuteAddon(ctx context.Context, req *implantpb.ExecuteAddon) (*clientpb.Task, error) {
	if session, err := getSession(ctx); err == nil {
		hasAddon := false
		for _, addon := range session.Addons {
			if addon.Name == req.Addon {
				hasAddon = true
				break
			}
		}
		if !hasAddon {
			return nil, errors.New("addon not found, please load_addon first")
		}
	}
	return rpc.GenericInternal(ctx, req, types.MsgBinaryResponse)
}
