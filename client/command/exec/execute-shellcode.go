package exec

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"google.golang.org/protobuf/proto"
	"os"
)

// ExecuteShellcodeCmd - Execute shellcode in-memory
func ExecuteShellcodeCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}

	//rwxPages := ctx.Flags.Bool("rwx-pages")
	//interactive := ctx.Flags.Bool("interactive")
	//if interactive {
	//	console.Log.Errorf("Interactive shellcode can only be executed in a session\n")
	//	return
	//}
	pid := ctx.Flags.Uint("pid")
	shellcodePath := ctx.Args.String("filepath")
	shellcodeBin, err := os.ReadFile(shellcodePath)
	if err != nil {
		console.Log.Errorf("%s\n", err.Error())
		return
	}
	//if pid != 0 && interactive {
	//	console.Log.Errorf("Cannot use both `--pid` and `--interactive`\n")
	//	return
	//}
	//shikataGaNai := ctx.Flags.Bool("shikata-ga-nai")
	//if shikataGaNai {
	//	if !rwxPages {
	//		console.Log.Errorf("Cannot use shikata ga nai without RWX pages enabled\n")
	//		return
	//	}
	//	arch := ctx.Flags.String("architecture")
	//	if arch != "386" && arch != "amd64" {
	//		console.Log.Errorf("Invalid shikata ga nai architecture (must be 386 or amd64)\n")
	//		return
	//	}
	//	iter := ctx.Flags.Int("iterations")
	//	console.Log.Infof("Encoding shellcode ...\n")
	//	resp, err := con.Rpc.ShellcodeEncoder(context.Background(), &clientpb.ShellcodeEncodeReq{
	//		Encoder:      clientpb.ShellcodeEncoder_SHIKATA_GA_NAI,
	//		Architecture: arch,
	//		Iterations:   uint32(iter),
	//		BadChars:     []byte{},
	//		Data:         shellcodeBin,
	//	})
	//	if err != nil {
	//		console.Log.Errorf("%s\n", err)
	//		return
	//	}
	//	oldSize := len(shellcodeBin)
	//	shellcodeBin = resp.GetData()
	//	console.Log.Infof("Shellcode encoded in %d iterations (%d bytes -> %d bytes)\n", iter, oldSize, len(shellcodeBin))
	//}
	//
	//if interactive {
	//	executeInteractive(ctx, ctx.Flags.String("process"), shellcodeBin, rwxPages, con)
	//	return
	//}
	//msg := fmt.Sprintf("Sending shellcode to %s ...", session.GetName())
	shellcodeTask, err := con.Rpc.ExecuteShellcode(con.ActiveTarget.Context(), &pluginpb.ExecuteShellcode{
		Bin:  shellcodeBin,
		Pid:  uint32(pid),
		Type: consts.ShellcodePlugin,
	})

	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}

	con.AddCallback(shellcodeTask.TaskId, func(msg proto.Message) {
		resp := msg.(*commonpb.Spite).GetAssemblyResponse()
		if resp.Err != "" {
			console.Log.Errorf("%s\n", resp.Err)
		} else {
			console.Log.Infof("Executed shellcode on target\n")
		}
	})
}
