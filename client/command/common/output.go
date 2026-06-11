package common

import (
	"fmt"
	"os"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// BindOutputFlags adds -f/--file flag to a command for saving output to a file.
func BindOutputFlags(cmd *cobra.Command) {
	Bind("output", false, cmd, func(f *pflag.FlagSet) {
		f.StringP("file", "f", "", "save output to file path")
	})
}

// HandleTaskOutput optionally saves task output to file based on -f flag,
// then always displays output to console.
func HandleTaskOutput(cmd *cobra.Command, con *core.Console, task *clientpb.Task) error {
	outputPath, _ := cmd.Flags().GetString("file")

	if outputPath == "" {
		con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
		return nil
	}

	if task == nil {
		return fmt.Errorf("task is nil")
	}

	session := con.GetInteractive()
	if session == nil {
		return fmt.Errorf("no active session")
	}

	taskReq := &clientpb.Task{
		SessionId: task.SessionId,
		TaskId:    task.TaskId,
		Need:      -1,
	}

	// Task content may arrive asynchronously. Wait for completion first so the
	// server-side cache/disk state is ready before collecting all chunks.
	if _, err := con.Rpc.WaitTaskFinish(session.Context(), taskReq); err != nil {
		return fmt.Errorf("failed to wait task finish: %w", err)
	}

	tasksContext, err := con.Rpc.GetAllTaskContent(session.Context(), taskReq)
	if err != nil {
		return fmt.Errorf("failed to get task content: %w", err)
	}

	var rendered []byte
	for _, spite := range tasksContext.Spites {
		eachTask := &clientpb.TaskContext{
			Task:    tasksContext.Task,
			Session: tasksContext.Session,
			Spite:   spite,
		}
		text, err := core.RenderTaskOutput(eachTask)
		if err != nil {
			return fmt.Errorf("failed to render task output: %w", err)
		}
		if text == "" {
			continue
		}
		rendered = append(rendered, []byte(text+"\n")...)
	}

	if err := os.WriteFile(outputPath, rendered, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	con.Log.Infof("Task output saved to: %s\n", outputPath)

	// File output already waited for completion, so a user-supplied --wait would
	// otherwise trigger a second wait/render pass in PersistentPostRunE.
	if waitFlag := cmd.Flags().Lookup("wait"); waitFlag != nil {
		_ = cmd.Flags().Set("wait", "false")
	}

	// Also display to console
	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}
