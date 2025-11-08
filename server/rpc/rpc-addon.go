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
	err := types.AssertRequestName(req, consts.ModuleListAddon)
	if err != nil {
		return nil, err
	}
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgListAddon, func(spite *implantpb.Spite) {
		if exts := spite.GetAddons(); exts != nil {
			sess, _ := getSession(ctx)
			sess.Addons = exts.Addons
		}
	})
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) LoadAddon(ctx context.Context, req *implantpb.LoadAddon) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerResponse(ch, types.MsgEmpty, func(spite *implantpb.Spite) {
		sess, _ := getSession(ctx)
		sess.Addons = append(sess.Addons, &implantpb.Addon{
			Name:   req.Name,
			Depend: req.Depend,
			Type:   req.Type,
		})
	})
	return greq.Task.ToProtobuf(), nil
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
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerResponse(ch, types.MsgBinaryResponse)
	return greq.Task.ToProtobuf(), nil
}
