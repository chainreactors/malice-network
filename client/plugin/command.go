package plugin

import (
	"github.com/spf13/cobra"
	"strings"
)

const CMDSeq = ":"

type Commands map[string]*Command

func (cs Commands) Commands() []*cobra.Command {
	var cmds []*cobra.Command
	for _, cmd := range cs {
		cmds = append(cmds, cmd.Command)
	}
	return cmds
}

func (cs Commands) Find(name string) *Command {
	subs := strings.Split(name, CMDSeq)
	if len(subs) == 0 {
		return nil
	}

	// 获取当前的子命令名
	subName := subs[0]

	// 检查当前命令是否存在，如果不存在则创建一个新的命令
	cmd, exists := cs[subName]
	if !exists {
		cmd = &Command{
			Name:    subName,
			Subs:    make(Commands),               // 初始化子命令映射
			Command: &cobra.Command{Use: subName}, // 创建对应的 Cobra 命令
		}
		cs[subName] = cmd
	}

	// 如果还有后续子命令，递归处理剩余的部分
	if len(subs) > 1 {
		return cmd.Subs.Find(strings.Join(subs[1:], CMDSeq))
	}

	// 如果已经到达最后一级，返回当前命令
	return cmd
}

func (cs Commands) SetCommand(name string, cmd *cobra.Command) {
	subs := strings.Split(name, CMDSeq)
	if len(subs) == 1 {
		cur := cs.Find(subs[0])
		cur.Command = cmd
		return
	}

	var currentCommands Commands = cs
	var parentCmd *Command

	for i := 0; i < len(subs); i++ {
		subName := subs[i]

		// 查找或创建当前级别的命令
		currentCmd := currentCommands.Find(subName)

		// 如果是最后一级，设置传入的 cobra.Command
		if i == len(subs)-1 {
			currentCmd.Command = cmd
		} else {
			// 如果不是最后一级，确保有 cobra.Command 用于添加子命令
			if currentCmd.Command == nil {
				currentCmd.Command = &cobra.Command{Use: currentCmd.Name}
			}
		}

		// 如果有父命令，将当前命令添加为父命令的子命令
		if parentCmd != nil && parentCmd.Command != nil {
			// 检查是否已经添加过，避免重复添加
			found := false
			for _, existingCmd := range parentCmd.Command.Commands() {
				if existingCmd.Use == currentCmd.Name {
					found = true
					break
				}
			}
			if !found {
				parentCmd.Command.AddCommand(currentCmd.Command)
			}
		}

		parentCmd = currentCmd
		currentCommands = currentCmd.Subs
	}
}

type Command struct {
	Name    string
	Long    string
	Example string
	Command *cobra.Command
	Subs    Commands
	Parent  *Command
}
