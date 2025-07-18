package taskschd

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
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

	session := con.GetInteractive()
	task, err := TaskSchdCreate(con.Rpc, session, name, path, triggerType, startBoundary)
	if err != nil {
		return err
	}

	session.Console(cmd, task, fmt.Sprintf("create scheduled task: %s", name))
	return nil
}

func TaskSchdCreate(rpc clientrpc.MaliceRPCClient, session *core.Session, name, path, triggerType, startBoundary string) (*clientpb.Task, error) {
	request := &implantpb.TaskScheduleRequest{
		Type: consts.ModuleTaskSchdCreate,
		Taskschd: &implantpb.TaskSchedule{
			Path:           "\\",
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
			"triggerType: trigger type for the task",
			"startBoundary: start boundary for the scheduled task",
		},
		[]string{"task"})
}
