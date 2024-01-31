package jobs

import (
	"context"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/styles"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/charmbracelet/bubbles/table"
	"golang.org/x/term"
	"strconv"
)

func Command(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: "jobs",
			Help: "List jobs",
			Flags: func(f *grumble.Flags) {
				f.String("k", "kill", "", "kill the designated job")
				f.Bool("K", "kill-all", false, "kill all the jobs")
				//f.String("f", "filter", "", "filter sessions by substring")
				//f.String("e", "filter-re", "", "filter sessions by regular expression")

				f.Int("t", "timeout", assets.DefaultSettings.DefaultTimeout, "command timeout in seconds")
			},
			Run: func(ctx *grumble.Context) error {
				JobCmd(ctx, con)
				return nil
			},
		},
	}
}
func JobCmd(ctx *grumble.Context, con *console.Console) {
	jobs, err := con.Rpc.GetJobs(context.Background(), &clientpb.Empty{})
	if err != nil {
		return
	}
	if len(jobs.Job) > 0 {
		printJobs(jobs, con)
	} else {
		console.Log.Info("No jobs")
	}

}

func printJobs(jobs *clientpb.Jobs, con *console.Console) {
	width, _, err := term.GetSize(0)
	var tableModel styles.TableModel
	var rowEntries []table.Row
	var row table.Row
	if err != nil {
		width = 99
	}
	if con.Settings.SmallTermWidth < width {
		tableModel = styles.TableModel{Columns: []table.Column{
			{Title: "ID", Width: 4},
			{Title: "name", Width: 4},
			{Title: "host", Width: 10},
			{Title: "port", Width: 5},
		}}
	}
	for _, job := range jobs.Job {
		row = table.Row{strconv.Itoa(int(job.Id)),
			job.Pipeline.GetTcp().Name,
			job.Pipeline.GetTcp().Host,
			strconv.Itoa(int(job.Pipeline.GetTcp().Port))}
		rowEntries = append(rowEntries, row)
	}
	tableModel.Rows = rowEntries
	tableModel.Run()
}
