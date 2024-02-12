package file

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
)

func DownloadCommand(con *console.Console) []*grumble.Command {
	return []*grumble.Command{{
		Name: "download",
		Help: "download file",
		Flags: func(f *grumble.Flags) {
			f.String("n", "name", "", "filename")
			f.String("p", "path", "", "filepath")
		},
		Run: func(ctx *grumble.Context) error {
			download(ctx, con)
			return nil
		},
	}}
}

func download(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}

	// TODO tui
	// 1. download 添加callback, 目前的代码无法获取下载进度
	// 2. 进度条添加参数, 是否直接显示在底栏， 如果不显示则只能在tasks中看到
	// 3. 写完后测试下效果是否正常

	//name := ctx.Flags.String("name")
	//path := ctx.Flags.String("path")

	//var download *clientpb.Task
	//var err error
	//ctrl := make(chan float64)
	//download, err = con.Rpc.Download(con.ActiveTarget.Context(), &pluginpb.DownloadRequest{
	//	Name: name,
	//	Path: path,
	//})
	//ctrl <- float64(download.Cur / download.Total)
	//go func() {
	//	m := tui.ProcessBarModel{
	//		Progress:        progress.New(progress.WithDefaultGradient()),
	//		ProgressPercent: <-ctrl,
	//	}
	//	tui.Run(m)
	//}()
	//if err != nil {
	//	console.Log.Errorf("")
	//}

}
