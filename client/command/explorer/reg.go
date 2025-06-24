package explorer

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/reg"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func regExplorerCmd(cmd *cobra.Command, con *repl.Console) error {
	rootPath := cmd.Flags().Arg(0)
	hive, path := reg.FormatRegPath(rootPath)
	session := con.GetInteractive()
	root := tui.TreeNode{
		Name: rootPath,
	}

	request := &implantpb.RegistryRequest{
		Type: consts.ModuleRegListKey,
		Registry: &implantpb.Registry{
			Hive: hive,
			Path: fileutils.FormatWindowPath(path),
		},
	}
	task, err := con.Rpc.RegListKey(session.Clone(consts.CalleeExplorer).Context(), request)
	if err != nil {
		return err
	}
	regChan := make(chan *implantpb.Response, 1)
	con.AddCallback(task, func(msg *clientpb.TaskContext) {
		regChan <- msg.Spite.GetResponse()
	})
	select {
	case resp := <-regChan:
		if len(resp.GetArray()) == 0 {
			con.Log.Info("No keys found")
			return nil
		}
		for _, key := range resp.GetArray() {
			root.Children = append(root.Children, &tui.TreeNode{
				Name: key,
			})
		}
		customDisplay := func(node *tui.TreeNode) string {
			result := fmt.Sprintf("Key: %s\n", node.Name)
			for _, info := range node.Info {
				result += fmt.Sprintf("\t%s\n", info)
			}
			return result
		}
		regModel, err := tui.NewTreeModel(root, customDisplay, tui.ChildrenTree)
		if err != nil {
			con.Log.Errorf("Error creating tree model: %v", err)
			return err
		}
		regModel = regModel.SetHeaderView(func(m *tui.TreeModel) string {
			return fmt.Sprintf("Current Path: %s%s\n", root.Name, regModel.Selected)
		})
		regModel = regModel.SetKeyBinding("enter", func(m *tui.TreeModel) (tea.Model, tea.Cmd) {
			return regEnterFuc(m, con)
		})
		regModel = regModel.SetKeyBinding("backspace", regBackFunc)
		err = regModel.Run()
		if err != nil {
			con.Log.Errorf("Error running explorer: %v", err)
			return err
		}
		tui.Reset()
	}
	return nil
}

func regEnterFuc(m *tui.TreeModel, con *repl.Console) (tea.Model, tea.Cmd) {
	selectedNode := m.Tree.Children[m.Cursor]
	if len(selectedNode.Children) > 0 {
		m.Selected = append(m.Selected, selectedNode.Name)
		m.Tree = selectedNode
		m.Cursor = 0
		return m, nil
	}
	session := con.GetInteractive()
	path := m.Root.Name + "\\" + selectedNode.Name
	hive, newPath := reg.FormatRegPath(path)
	keyTask, err := con.Rpc.RegListKey(session.Clone(consts.CalleeExplorer).Context(), &implantpb.RegistryRequest{
		Type: consts.ModuleRegListKey,
		Registry: &implantpb.Registry{
			Hive: hive,
			Path: newPath,
		},
	})
	if err != nil {
		con.Log.Errorf("Error listing keys: %v", err)
		return m, nil
	}
	valueTask, err := con.Rpc.RegListValue(session.Clone(consts.CalleeExplorer).Context(), &implantpb.RegistryRequest{
		Type: consts.ModuleRegListValue,
		Registry: &implantpb.Registry{
			Hive: hive,
			Path: newPath,
		},
	})
	if err != nil {
		con.Log.Errorf("Error listing values: %v", err)
		return m, nil
	}
	regKeyChan := make(chan *implantpb.Response, 1)
	regValueChan := make(chan *implantpb.Response, 1)
	con.AddCallback(keyTask, func(msg *clientpb.TaskContext) {
		regKeyChan <- msg.Spite.GetResponse()
	})
	con.AddCallback(valueTask, func(msg *clientpb.TaskContext) {
		regValueChan <- msg.Spite.GetResponse()
	})
	select {
	case keyResp := <-regKeyChan:
		for _, key := range keyResp.GetArray() {
			selectedNode.Children = append(selectedNode.Children, &tui.TreeNode{
				Name: key,
			})
		}
	case valueResp := <-regValueChan:
		for k, v := range valueResp.GetKv() {
			selectedNode.Info = append(selectedNode.Info, fmt.Sprintf("Value: %s | Data: %s", k, v))
		}
	}
	if len(selectedNode.Children) > 0 {
		m.Selected = append(m.Selected, selectedNode.Name)
		m.Tree = selectedNode
		m.Cursor = 0
		return m, nil
	}
	return m, nil
}

func regBackFunc(m *tui.TreeModel) (tea.Model, tea.Cmd) {
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
