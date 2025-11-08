package taskschd

import (
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
	"strings"
)

// TaskSchdCreateCmd creates a new scheduled task.
func TaskSchdCreateCmd(cmd *cobra.Command, con *repl.Console) error {
	// 内嵌的 Flag 解析
	name, _ := cmd.Flags().GetString("name")
	path, _ := cmd.Flags().GetString("path")
	triggerType, _ := cmd.Flags().GetString("trigger_type")
	startBoundary, _ := cmd.Flags().GetString("start_boundary")
	taskFolder, _ := cmd.Flags().GetString("task_folder")
	session := con.GetInteractive()
	task, err := TaskSchdCreate(con.Rpc, session, name, path, taskFolder, triggerType, startBoundary)
	if err != nil {
		return err
	}

	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func TaskSchdCreate(rpc clientrpc.MaliceRPCClient, session *client.Session, name, path, taskFolder, triggerType, startBoundary string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdCreate,
		Taskschd: &implantpb.TaskSchedule{
			Path:           taskFolder,
			Name:           name,
			ExecutablePath: path,
			//TriggerType:    triggerType,
			StartBoundary: startBoundary,
		},
	}
	switch strings.ToLower(triggerType) {
	case "daily", "day":
		request.Taskschd.TriggerType = 2
	case "weekly", "week":
		request.Taskschd.TriggerType = 3
	case "monthly", "month", "mon":
		request.Taskschd.TriggerType = 4
	case "atlogon", "logon":
		request.Taskschd.TriggerType = 9
	case "start", "startup":
		request.Taskschd.TriggerType = 8
	}
	return rpc.TaskSchdCreate(session.Context(), request)
}

func RegisterTaskSchdCreateFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleTaskSchdCreate,
		TaskSchdCreate,
		"",
		nil,
		output.ParseStatus,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModuleTaskSchdCreate,
		consts.ModuleTaskSchdCreate,
		//session *core.Session, namespace string, args []string
		consts.ModuleTaskSchdCreate+`(active(), "task_name", "process_path", 1, "2023-10-10T09:00:00")`,
		[]string{
			"sess: special session",
			"name: name of the scheduled task",
			"path: path to the executable for the scheduled task",
			"task_folder: task folder for the scheduled task",
			"triggerType: trigger type for the task",
			"startBoundary: start boundary for the scheduled task",
		},
		[]string{"task"})
}
