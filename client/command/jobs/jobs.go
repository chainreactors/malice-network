package jobs

import (
	"context"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
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
