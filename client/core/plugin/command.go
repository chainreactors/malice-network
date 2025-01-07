package plugin

import (
	"github.com/spf13/cobra"
	"strings"
)

const CMDSeq = ":"

type Commands map[string]*Command

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
			Name: subName,
			Subs: make(Commands),               // 初始化子命令映射
			CMD:  &cobra.Command{Use: subName}, // 创建对应的 Cobra 命令
		}
		cs[subName] = cmd
	}

	// 如果还有后续子命令，递归处理剩余的部分
	if len(subs) > 1 {
		// 递归查找或创建剩余的子命令
		return cmd.Subs.Find(strings.Join(subs[1:], CMDSeq))
	}

	// 如果已经到达最后一级，返回当前命令
	return cmd
}

func (cs Commands) SetCommand(name string, cmd *cobra.Command) {
	subs := strings.Split(name, CMDSeq)
	if len(subs) == 1 {
		cur := cs.Find(subs[0])
		cur.CMD = cmd
		return
	}

	// 遍历每一级，查找或创建各级命令
	var parentCmd *Command
	for i := 0; i < len(subs)-1; i++ {
		currentName := strings.Join(subs[:i+1], CMDSeq)
		if parentCmd == nil {
			// 查找或创建第一级命令
			parentCmd = cs.Find(currentName)
		} else {
			// 查找或创建后续的子命令
			parentCmd = parentCmd.Subs.Find(subs[i])
		}

		if parentCmd.CMD == nil {
			parentCmd.CMD = &cobra.Command{Use: parentCmd.Name}
		}
	}

	// 处理最后一级命令
	finalCmdName := subs[len(subs)-1]
	finalCmd := parentCmd.Subs.Find(finalCmdName)
	if finalCmd == nil {
		finalCmd = &Command{
			Name: finalCmdName,
			CMD:  cmd, // 最后一级命令使用传入的 cmd
			Subs: make(Commands),
		}
		parentCmd.Subs[finalCmdName] = finalCmd
	} else {
		finalCmd.CMD = cmd
	}

	// 将最后一级命令添加为父级命令的子命令
	if parentCmd != nil && parentCmd.CMD != nil {
		parentCmd.CMD.AddCommand(cmd)
	}
}

type Command struct {
	Name    string
	Long    string
	Example string
	CMD     *cobra.Command
	Subs    Commands
	Parent  *Command
}
