package explorer

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/tui"
	tea "github.com/charmbracelet/bubbletea"
	"strconv"
	//"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func taskschdExplorerCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	root := tui.TreeNode{
		Name: "Task Scheduler",
	}

	task, err := con.Rpc.TaskSchdList(session.Clone(consts.CalleeExplorer).Context(), &implantpb.Request{
		Name: consts.ModuleTaskSchdList,
	})
	if err != nil {
		return err
	}
	taskschdChan := make(chan []*implantpb.TaskSchedule, 1)
	con.AddCallback(task, func(msg *clientpb.TaskContext) {
		resp := msg.Spite.GetSchedulesResponse()
		taskschdChan <- resp.GetSchedules()
	})
	select {
	case taskschs := <-taskschdChan:
		for _, protoTaskSch := range taskschs {
			newTasksch := &tui.TreeNode{
				Name: "Name: " + protoTaskSch.GetName(),
				Info: []string{"Enable: " + strconv.FormatBool(protoTaskSch.GetEnabled()),
					"Path: " + protoTaskSch.GetPath(),
					"Executable Path: " + protoTaskSch.GetExecutablePath(),
					"Start Boundary: " + protoTaskSch.GetStartBoundary(),
					"Trigger Type: " + strconv.FormatInt(int64(protoTaskSch.GetTriggerType()), 10),
					"Description: " + protoTaskSch.GetDescription(),
					"Last RunTime: " + protoTaskSch.GetLastRunTime(),
					"Next RunTime: " + protoTaskSch.GetNextRunTime(),
				},
			}
			root.Children = append(root.Children, newTasksch)
		}
		customDisplay := func(node *tui.TreeNode) string {
			return formatNodeInfo(node, 0)
		}
		taskschdModel, err := tui.NewTreeModel(root, customDisplay, tui.InfoTree)
		if err != nil {
			return err
		}
		taskschdModel = taskschdModel.SetHeaderView(func(m *tui.TreeModel) string {
			return "taskched"
		})
		taskschdModel = taskschdModel.SetKeyBinding("enter", func(m *tui.TreeModel) (tea.Model, tea.Cmd) {
			return taskEnterFunc(m, con)
		})
		taskschdModel = taskschdModel.SetKeyBinding("backspace", taskBackFunc)
		taskschdModel.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func taskEnterFunc(m *tui.TreeModel, con *repl.Console) (tea.Model, tea.Cmd) {
	selectedNode := m.Tree.Children[m.Cursor]
	m.Selected = append(m.Selected, selectedNode.Name)
	return m, nil
}

func taskBackFunc(m *tui.TreeModel) (tea.Model, tea.Cmd) {
	if m.Cursor >= 0 && m.Cursor < len(m.Tree.Children) {
		// Get the name of the node at the current cursor position
		selectedNode := m.Tree.Children[m.Cursor]
		nodeName := selectedNode.Name

		// Find the index of nodeName in m.Selected
		index := -1
		for i, name := range m.Selected {
			if name == nodeName {
				index = i
				break
			}
		}

		// If nodeName is found in m.Selected, remove it
		if index != -1 {
			m.Selected = append(m.Selected[:index], m.Selected[index+1:]...)
		}
	}
	return m, nil
}

// Helper function to format and display the node's info with indentation and prefix
func formatNodeInfo(node *tui.TreeNode, level int) string {
	// Build indentation prefix based on level
	indent := ""
	for i := 0; i < level; i++ {
		indent += "|   "
	}

	// Format the node name with indentation and prefix
	result := fmt.Sprintf("%s\n", node.Name)

	// Loop through and format each info item with additional indentation
	for _, info := range node.Info {
		result += fmt.Sprintf("%s|   %s\n", indent, info)
	}

	return result
}
