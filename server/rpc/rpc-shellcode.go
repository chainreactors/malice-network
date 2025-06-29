package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/internal/generate"
	"github.com/wabzsy/gonut"
	"path/filepath"
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

func (rpc *Server) MaleficSRDI(ctx context.Context, req *clientpb.Builder) (*clientpb.Artifact, error) {
	var filePath, realName string
	var err error
	var artifact *models.Builder
	var bin []byte
	target, ok := consts.GetBuildTarget(req.Target)
	if !ok {
		return nil, errs.ErrInvalidateTarget
	}
	if req.Id != 0 {
		builder, err := db.GetArtifactById(req.Id)
		if err != nil {
			return nil, err
		}
		bin, err = build.SRDIArtifact(builder, target.OS, target.Arch)
		artifact = builder
		if err != nil {
			return nil, err
		}
	} else {
		dst := encoders.UUID()
		filePath = filepath.Join(configs.TempPath, dst)
		realName = req.Name
		err = build.SaveArtifact(dst, req.Bin)
		artifact, bin, err = build.NewMaleficSRDIArtifact(realName, req.Type, filePath, target.OS, target.Arch, req.Stage, req.FunctionName, req.UserDataPath)
		if err != nil {
			return nil, err
		}
	}

	return artifact.ToArtifact(bin), nil
}
