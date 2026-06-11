package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func (rpc *Server) ListAddon(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	return rpc.AssertAndHandleWithSession(ctx, req, consts.ModuleListAddon, types.MsgListAddon, func(greq *GenericRequest, spite *implantpb.Spite) {
		applyAddonsResponse(greq.Session, spite, false)
	})
}

func (rpc *Server) LoadAddon(ctx context.Context, req *implantpb.LoadAddon) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	return rpc.GenericInternalWithSession(ctx, req, types.MsgEmpty, func(greq *GenericRequest, spite *implantpb.Spite) {
		applyAddonLoad(greq.Session, req)
	})
}

// applyAddonsResponse replaces the session's addon list with deduplicated
// addons from the response spite, then persists. If replace is true the
// existing list is cleared first; otherwise new addons are merged in.
func applyAddonsResponse(sess *core.Session, spite *implantpb.Spite, replace bool) {
	exts := spite.GetAddons()
	if exts == nil {
		return
	}
	if replace {
		sess.Addons = nil
	}
	seen := make(map[string]bool, len(sess.Addons))
	for _, a := range sess.Addons {
		if a != nil && a.GetName() != "" {
			seen[a.GetName()] = true
		}
	}
	for _, a := range exts.GetAddons() {
		if a == nil || a.GetName() == "" {
			continue
		}
		if seen[a.GetName()] {
			continue
		}
		seen[a.GetName()] = true
		sess.Addons = append(sess.Addons, a)
	}
	sess.SaveAndNotify("")
}

// applyAddonLoad adds or updates a single addon on the session, deduplicating
// by name. If the addon already exists its metadata is refreshed.
func applyAddonLoad(sess *core.Session, req *implantpb.LoadAddon) {
	if req == nil || req.GetName() == "" {
		return
	}
	for _, a := range sess.Addons {
		if a != nil && a.GetName() == req.GetName() {
			a.Type = req.GetType()
			a.Depend = req.GetDepend()
			sess.SaveAndNotify("")
			return
		}
	}
	sess.Addons = append(sess.Addons, &implantpb.Addon{
		Name:   req.GetName(),
		Depend: req.GetDepend(),
		Type:   req.GetType(),
	})
	sess.SaveAndNotify("")
}

func (rpc *Server) ExecuteAddon(ctx context.Context, req *implantpb.ExecuteAddon) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
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
