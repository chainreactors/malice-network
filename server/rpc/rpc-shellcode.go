package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/generate"
	"os"
	"path/filepath"
)

func (rpc *Server) EXE2Shellcode(ctx context.Context, req *clientpb.EXE2Shellcode) (*clientpb.Bin, error) {
	if req.Type == "donut" {
		bin, err := generate.DonutShellcodeFromPE(req.Bin, req.Arch, false, req.Params, "", "", false, false, true)
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
		bin, err := generate.DonutShellcodeFromPE(req.Bin, req.Arch, false, req.Params, "", "", true, false, true)
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

func (rpc *Server) MaleficSRDI(ctx context.Context, req *clientpb.MutantFile) (*clientpb.Bin, error) {
	if req.Id != "" {
		filePath, err := build.GetOutPutPath(req.Id)
		if err != nil {
			return nil, err
		}
		fileName := filepath.Base(filePath)
		bin, err := build.MaleficSRDI(req, filePath, filepath.Join(configs.SRDIOutputPath, fileName))
		if err != nil {
			return nil, err
		}
		return &clientpb.Bin{Bin: bin, Name: fileName}, nil
	}
	srcPath := filepath.Join(configs.BuildOutputPath, req.Name)
	dstPath := filepath.Join(configs.SRDIOutputPath, filepath.Ext(req.Name))
	err := os.WriteFile(srcPath, req.Bin, 0644)
	if err != nil {
		return nil, err
	}
	bin, err := build.MaleficSRDI(req, srcPath, dstPath)
	if err != nil {
		return nil, err
	}
	return &clientpb.Bin{Bin: bin}, nil
}
