package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
)

func (rpc *Server) ListExtensions(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerAsyncResponse(ch, types.MsgListExtension, func(spite *implantpb.Spite) {
		if exts := spite.GetExtensions(); exts != nil {
			sess, _ := getSession(ctx)
			sess.Extensions = exts
		}
	})
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) LoadExtension(ctx context.Context, req *implantpb.LoadExtension) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	go greq.HandlerAsyncResponse(ch, types.MsgEmpty, func(spite *implantpb.Spite) {
		sess, _ := getSession(ctx)
		sess.Extensions.Extensions = append(sess.Extensions.Extensions, &implantpb.Extension{
			Name:   req.Name,
			Depend: req.Depend,
			Type:   req.Type,
		})
	})
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) ExecuteExtension(ctx context.Context, req *implantpb.ExecuteExtension) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	ch, err := rpc.asyncGenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	go greq.HandlerAsyncResponse(ch, types.MsgAssemblyResponse)
	return greq.Task.ToProtobuf(), nil
}
