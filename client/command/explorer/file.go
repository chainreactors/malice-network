package explorer

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func fileExplorerCmd(cmd *cobra.Command, con *repl.Console) {
	session := con.GetInteractive()
	root := tui.TreeNode{
		Name: "/",
	}

	task, err := con.Rpc.Ls(session.Clone(consts.CalleeExplorer).Context(), &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: "/",
	})
	if err != nil {
		con.Log.Errorf("load directory error: %v\n", err)
		return
	}
	session.Console(cmd, task, "")
	fileChan := make(chan []*implantpb.FileInfo, 1)
	con.AddCallback(task, func(msg *clientpb.TaskContext) {
		resp := msg.Spite.GetLsResponse()
		fileChan <- resp.GetFiles()
	})
	select {
	case files := <-fileChan:
		for _, protoFile := range files {
			root.Children = append(root.Children, &tui.TreeNode{
				Name: protoFile.GetName(),
				Info: []string{
					strconv.FormatBool(protoFile.IsDir),
					formatFileMode(protoFile.Mode),
					strconv.FormatUint(protoFile.Size, 10),
					strconv.FormatInt(protoFile.ModTime, 10)},
			})
		}
		customDisplay := func(node *tui.TreeNode) string {
			timestamp, err := strconv.ParseInt(node.Info[3], 10, 64)
			var timeStr string
			var isFile string
			if err == nil {
				timeStr = time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
			} else {
				timeStr = node.Info[3]
			}
			if node.Info[0] == "true" {
				isFile = "dir"
			} else {
				isFile = node.Info[2]
			}

			rawInfo := node.Info[1]
			rawName := node.Name
			rawFile := isFile
			rawTime := timeStr

			formatted := fmt.Sprintf("%-12s %-30s %-10s %-20s",
				padRight(rawInfo, 12),
				padRight(rawName, 30),
				padRight(rawFile, 10),
				padRight(rawTime, 20))

			formatted = strings.Replace(formatted, rawInfo, tui.GrayFg.Render(rawInfo), 1)
			formatted = strings.Replace(formatted, rawName, tui.SlateBlueFg.Render(rawName), 1)
			formatted = strings.Replace(formatted, rawFile, tui.DarkGrayFg.Render(rawFile), 1)
			formatted = strings.Replace(formatted, rawTime, tui.DarkGrayFg.Render(rawTime), 1)

			return formatted
		}

		fileModel, err := tui.NewTreeModel(root, customDisplay, tui.ChildrenTree)
		if err != nil {
			con.Log.Errorf("Error creating tree model: %v", err)
			return
		}
		fileModel = fileModel.SetHeaderView(func(m *tui.TreeModel) string {
			return fmt.Sprintf("Current Path: %s%s\n", root.Name, filepath.Join(m.Selected...))
		})
		// Register custom action for 'enter' key
		fileModel = fileModel.SetKeyBinding("enter", func(m *tui.TreeModel) (tea.Model, tea.Cmd) {
			return fileEnterFunc(cmd, m, con)
		})
		fileModel = fileModel.SetKeyBinding("backspace", backFunc)
		fileModel = fileModel.SetKeyBinding("r", func(m *tui.TreeModel) (tea.Model, tea.Cmd) {
			return freshFunc(m, con)
		})
		fileModel.Run()
		break
	}
}
func formatFileMode(mode uint32) string {
	var permissions = []rune{'-', '-', '-', '-', '-', '-', '-', '-', '-'}
	if mode&0400 != 0 {
		permissions[0] = 'r'
	}
	if mode&0200 != 0 {
		permissions[1] = 'w'
	}
	if mode&0100 != 0 {
		permissions[2] = 'x'
	}
	if mode&0040 != 0 {
		permissions[3] = 'r'
	}
	if mode&0020 != 0 {
		permissions[4] = 'w'
	}
	if mode&0010 != 0 {
		permissions[5] = 'x'
	}
	if mode&0004 != 0 {
		permissions[6] = 'r'
	}
	if mode&0002 != 0 {
		permissions[7] = 'w'
	}
	if mode&0001 != 0 {
		permissions[8] = 'x'
	}
	return string(permissions)
}

// Helper function to pad strings to the desired width
func padRight(str string, length int) string {
	return fmt.Sprintf("%-*s", length, str)
}

func fileEnterFunc(cmd *cobra.Command, m *tui.TreeModel, con *repl.Console) (tea.Model, tea.Cmd) {
	selectedNode := m.Tree.Children[m.Cursor]
	session := con.GetInteractive()
	if len(selectedNode.Children) > 0 {
		m.Selected = append(m.Selected, selectedNode.Name)
		m.Tree = selectedNode
		m.Cursor = 0
		return m, nil
	}
	if selectedNode.Info[0] == "false" {
		return m, nil
	}
	path := filepath.Join(m.Selected...)
	task, err := con.Rpc.Ls(session.Clone(consts.CalleeExplorer).Context(), &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: filepath.Join(m.Root.Name, path, selectedNode.Name),
	})
	session.Console(cmd, task, "")
	if err != nil {
		con.Log.Errorf("load directory error: %v\n", err)
		return m, nil
	}
	fileChan := make(chan []*implantpb.FileInfo, 1)
	con.AddCallback(task, func(msg *clientpb.TaskContext) {
		resp := msg.Spite.GetLsResponse()
		fileChan <- resp.GetFiles()
	})
	select {
	case files := <-fileChan:
		if len(files) == 0 {
			return m, nil
		}
		for _, protoFile := range files {
			selectedNode.Children = append(selectedNode.Children, &tui.TreeNode{
				Name: protoFile.GetName(),
				Info: []string{
					strconv.FormatBool(protoFile.IsDir),
					formatFileMode(protoFile.Mode),
					strconv.FormatUint(protoFile.Size, 10),
					strconv.FormatInt(protoFile.ModTime, 10)},
			})
		}
	}
	m.Selected = append(m.Selected, selectedNode.Name)
	fmt.Printf("Selected: %v\n", m.Selected)
	m.Tree = selectedNode
	m.Cursor = 0
	tui.ClearAll()
	return m, nil
}

func backFunc(m *tui.TreeModel) (tea.Model, tea.Cmd) {
	if len(m.Selected) > 0 {
		m.Selected = m.Selected[:len(m.Selected)-1]
		// Navigate back to the root and go down to the correct path
		m.Tree = m.Root
		for _, part := range m.Selected {
			for _, child := range m.Tree.Children {
				if child.Name == part {
					m.Tree = child
					break
				}
			}
		}
		m.Cursor = 0
	}
	return m, nil
}

func freshFunc(m *tui.TreeModel, con *repl.Console) (tea.Model, tea.Cmd) {
	selectedNode := m.Tree
	selectedNode.Children = []*tui.TreeNode{}
	session := con.GetInteractive()
	task, err := con.Rpc.Ls(session.Clone(consts.CalleeExplorer).Context(), &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: "./" + selectedNode.Name,
	})
	if err != nil {
		con.Log.Errorf("load directory error: %v\n", err)
		return m, nil
	}
	fileChan := make(chan []*implantpb.FileInfo, 1)
	con.AddCallback(task, func(msg *clientpb.TaskContext) {
		resp := msg.Spite.GetLsResponse()
		fileChan <- resp.GetFiles()
	})
	select {
	case files := <-fileChan:
		for _, protoFile := range files {
			selectedNode.Children = append(selectedNode.Children, &tui.TreeNode{
				Name: protoFile.GetName(),
				Info: []string{
					strconv.FormatBool(protoFile.IsDir),
					formatFileMode(protoFile.Mode),
					strconv.FormatUint(protoFile.Size, 10),
					strconv.FormatInt(protoFile.ModTime, 10)},
			})
		}
	}
	m.Tree = selectedNode
	m.Cursor = 0
	return m, nil
}
