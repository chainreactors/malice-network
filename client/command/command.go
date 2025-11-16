package command

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

// Bind is a convenience function to bind flags to a given command.
// name - The name of the flag set (can be empty).
// cmd  - The command to which the flags should be bound.

type BindFunc func(group string, cmds ...func(con *core.Console) []*cobra.Command)

func MakeBind(cmd *cobra.Command, con *core.Console) BindFunc {
	return func(group string, cmds ...func(con *core.Console) []*cobra.Command) {
		found := false

		// Ensure the given command group is available in the menu.
		if group != "" {
			for _, grp := range cmd.Groups() {
				if grp.Title == group {
					found = true
					break
				}
			}

			if !found {
				cmd.AddGroup(&cobra.Group{
					ID:    group,
					Title: group,
				})
			}
		}

		// Bind the command to the root
		for _, command := range cmds {
			for _, c := range command(con) {
				if c.Annotations == nil {
					c.Annotations = map[string]string{}
				}
				c.GroupID = group
				c.Annotations["menu"] = cmd.Name()
				updateCommand(con, c, group)
				cmd.AddCommand(c)
			}
		}
	}
}

func updateCommand(con *core.Console, c *cobra.Command, group string) {
	c.SetHelpFunc(help.HelpFunc)
	c.SetUsageFunc(help.UsageFunc)
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	if c.Annotations["opsec"] != "" {
		c.PreRunE = func(cmd *cobra.Command, args []string) error {
			err := common.OpsecConfirm(cmd)
			if err != nil {
				return err
			}
			return nil
		}
	}

	// 根据 "static" annotation 自动添加 --static flag
	if c.Annotations["static"] != "" {
		// 检查是否已经定义了 static flag
		if c.Flags().Lookup("static") == nil {
			c.Flags().BoolP("static", "s", false, "non-interactive mode")
		}
	}

	con.CMDs[c.Name()] = c
	if dep, ok := c.Annotations["depend"]; ok {
		con.Helpers[dep] = c
	}

	for _, subCmd := range c.Commands() {
		updateCommand(con, subCmd, group)
	}
}
