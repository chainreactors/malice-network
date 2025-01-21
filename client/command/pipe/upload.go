package pipe

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/chainreactors/utils/encode"
	"github.com/spf13/cobra"
)

// PipeUploadCmd uploads a file's content to a named pipe.
func PipeUploadCmd(cmd *cobra.Command, con *repl.Console) error {
	pipe := cmd.Flags().Arg(0)
	path := cmd.Flags().Arg(1)

	task, err := PipeUpload(con.Rpc, con.GetInteractive(), pipe, path)
	if err != nil {
		return err
	}

	con.GetInteractive().Console(task, fmt.Sprintf("Uploaded file %s to pipe %s", path, pipe))
	return nil
}

func PipeUpload(rpc clientrpc.MaliceRPCClient, session *core.Session, pipe string, path string) (*clientpb.Task, error) {
	data, err := pe.Unpack(path)
	if err != nil {
		core.Log.Errorf("Can't open file: %s", err)
		return nil, err
	}

	task, err := rpc.PipeUpload(session.Context(), &implantpb.PipeRequest{
		Type: consts.ModulePipeUpload,
		Pipe: &implantpb.Pipe{
			Name: fileutils.FormatWindowPath(pipe),
			Data: data,
		},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func PipeUploadRaw(rpc clientrpc.MaliceRPCClient, session *core.Session, pipe, data string) (*clientpb.Task, error) {
	task, err := rpc.PipeUpload(session.Context(), &implantpb.PipeRequest{
		Type: consts.ModulePipeUpload,
		Pipe: &implantpb.Pipe{
			Name: fileutils.FormatWindowPath(pipe),
			Data: []byte(data),
		},
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

// 注册 PipeUpload 命令
func RegisterPipeUploadFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModulePipeUpload,
		PipeUpload,
		"",
		nil,
		common.ParseStatus,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModulePipeUpload,
		consts.ModulePipeUpload,
		consts.ModulePipeUpload+`(active(), "pipe_name", "file_path")`,
		[]string{"session: special session",
			"pipe: target pipe",
			"path: file path to upload",
		},
		[]string{"task"})

	con.RegisterImplantFunc("pipe_upload_raw",
		func(rpc clientrpc.MaliceRPCClient, session *core.Session, pipe string, data string) (*clientpb.Task, error) {
			return PipeUpload(rpc, session, pipe, fmt.Sprintf("bin:%s", encode.Base64Encode([]byte(data))))
		},
		"",
		nil,
		common.ParseStatus,
		nil)

}
