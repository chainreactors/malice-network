package jobs

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/desertbit/grumble"
)

func JobCmd(ctx *grumble.Context, con *console.Console) {
	jobs, err := con.Rpc.GetJobs(context.Background(), &clientpb.Empty{})
	if err != nil {
		return
	}
	printJobs(jobs)
}

func printJobs(jobs *clientpb.Jobs) {
	fmt.Println(jobs)
}
