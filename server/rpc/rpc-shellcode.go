package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/generate"
	"github.com/wabzsy/gonut"
)

func (rpc *Server) EXE2Shellcode(ctx context.Context, req *clientpb.EXE2Shellcode) (*clientpb.Bin, error) {
	if req.Type == "donut" {
		bin, err := gonut.DonutShellcodeFromPE("1.exe", req.Bin, req.Arch, req.Params, false, true)
		if err != nil {
			return nil, err
		}
		return &clientpb.Bin{Bin: bin}, nil
	} else {
		return nil, fmt.Errorf("unknown type")
	}
}

func (rpc *Server) DLL2Shellcode(ctx context.Context, req *clientpb.DLL2Shellcode) (*clientpb.Bin, error) {
	if req.Type == "donut" {
		bin, err := gonut.DonutShellcodeFromPE("1.dll", req.Bin, req.Arch, req.Params, false, true)
		if err != nil {
			return nil, err
		}
		return &clientpb.Bin{Bin: bin}, nil
	} else if req.Type == "srdi" {
		bin, err := generate.ShellcodeRDIFromBytes(req.Bin, req.Entrypoint, req.Params)
		if err != nil {
			return nil, err
		}
		return &clientpb.Bin{Bin: bin}, nil
	} else {
		return nil, fmt.Errorf("unknown type")
	}
}

func (rpc *Server) ShellcodeEncode(ctx context.Context, req *clientpb.ShellcodeEncode) (*clientpb.Bin, error) {
	if req.Type == "sgn" {
		bin, err := generate.EncodeShellcode(req.Shellcode, req.Arch, int(req.Iterations), []byte{})
		if err != nil {
			return nil, err
		}
		return &clientpb.Bin{Bin: bin}, nil
	} else {
		return nil, fmt.Errorf("unknown type")
	}
}
