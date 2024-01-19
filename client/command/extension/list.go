package extension

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
)

// ExtensionsListCmd - List all extension loaded on the active session/beacon
func ExtensionsListCmd(ctx *grumble.Context, con *console.Console) {
	session := con.ActiveTarget.GetInteractive()
	if session == nil {
		return
	}

	//extList, err := con.Rpc.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
	//	Request: con.ActiveTarget.Request(ctx),
	//})
	//if err != nil {
	//	console.Log.Errorf("%s\n", err)
	//	return
	//}

	//if extList.Response != nil && extList.Response.Err != "" {
	//	console.Log.Errorf("%s\n", extList.Response.Err)
	//	return
	//}
	//if len(extList.Names) > 0 {
	//	console.Log.Infof("Loaded extensions:\n")
	//	for _, ext := range extList.Names {
	//		console.Log.Infof("- %s\n", ext)
	//	}
	//}
}
