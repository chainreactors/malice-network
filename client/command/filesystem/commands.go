package filesystem

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strconv"
	"strings"
	"time"
)

func Commands(con *repl.Console) []*cobra.Command {
	pwdCmd := &cobra.Command{
		Use:   consts.ModulePwd,
		Short: "Print working directory",
		Long:  "print working directory in implant",
		Run: func(cmd *cobra.Command, args []string) {
			PwdCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleCat,
		},
	}

	catCmd := &cobra.Command{
		Use:   consts.ModuleCat + " [implant_file]",
		Short: "Print file content",
		Long:  "concatenate and display the contents of file in implant",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CatCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleCat,
		},
		Example: `~~~
cat file.txt			
~~~`,
	}

	common.BindArgCompletions(catCmd, nil,
		carapace.ActionValues().Usage("cat file name"))

	cdCmd := &cobra.Command{
		Use:   consts.ModuleCd,
		Short: "Change directory",
		Long:  "change the shell's current working directory in implant",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CdCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleLs,
		},
	}
	common.BindArgCompletions(cdCmd, nil,
		carapace.ActionValues().Usage("cd path"))

	chmodCmd := &cobra.Command{
		Use:   consts.ModuleChmod + " [file] [mode]",
		Short: "Change file mode",
		Long:  "change the permissions of files and directories in implant",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ChmodCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"os":     "linux,mac",
			"depend": consts.ModuleChmod,
		},
		Example: `~~~
chmod ./file.txt 644
~~~`,
	}

	common.BindArgCompletions(chmodCmd, nil,
		carapace.ActionValues().Usage("chmod file mode"),
		carapace.ActionValues().Usage("chmod file path"))

	chownCmd := &cobra.Command{
		Use:   consts.ModuleChown + " [file] [user]",
		Short: "Change file owner",
		Long:  "change the ownership of a file or directory in implant",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ChownCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"os":     "linux,mac",
			"depend": consts.ModuleChown,
		},
		Example: `~~~
chown user ./file.txt 
~~~`,
	}

	common.BindArgCompletions(chownCmd, nil,
		carapace.ActionValues().Usage("chown user"),
		carapace.ActionValues().Usage("chown file path"))

	common.BindFlag(chownCmd, func(f *pflag.FlagSet) {
		f.BoolP("recursive", "r", false, "recursive")
		f.StringP("gid", "g", "", "Group id")
	})

	cpCmd := &cobra.Command{
		Use:   consts.ModuleCp + " [source] [target]",
		Short: "Copy file",
		Long:  "copy files and directories in implant",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			CpCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleCp,
		},
		Example: `~~~
cp /tmp/file.txt /tmp/file2.txt 
~~~`,
	}

	common.BindArgCompletions(cpCmd, nil,
		carapace.ActionValues().Usage("source file"),
		carapace.ActionValues().Usage("target file"))

	lsCmd := &cobra.Command{
		Use:   consts.ModuleLs + " [path]",
		Short: "List directory",
		Long:  "list directory contents in implant",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			LsCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleLs,
		},
		Example: `~~~
ls /tmp	
~~~`,
	}

	common.BindArgCompletions(lsCmd, nil,
		carapace.ActionValues().Usage("ls path"))

	mkdirCmd := &cobra.Command{
		Use:   consts.ModuleMkdir + " [path]",
		Short: "Make directory",
		Long:  "make directories in implant",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			MkdirCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleMkdir,
		},
		Example: `~~~
mkdir /tmp
~~~`,
	}

	common.BindArgCompletions(mkdirCmd, nil,
		carapace.ActionValues().Usage("mkdir path"))

	mvCmd := &cobra.Command{
		Use:   consts.ModuleMv + " [source] [target]",
		Short: "Move file",
		Long:  "move files and directories in implant",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			MvCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleMv,
		},
		Example: `~~~
mv /tmp/file.txt /tmp/file2.txt
~~~`,
	}

	common.BindArgCompletions(mvCmd, nil,
		carapace.ActionValues().Usage("source file"),
		carapace.ActionValues().Usage("target file"))

	rmCmd := &cobra.Command{
		Use:   consts.ModuleRm + " [file]",
		Short: "Remove file",
		Long:  "remove files and directories in implant",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RmCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRm,
		},
		Example: `~~~
rm /tmp/file.txt
~~~`,
	}

	common.BindArgCompletions(rmCmd, nil,
		carapace.ActionValues().Usage("rm file name"))

	return []*cobra.Command{
		pwdCmd,
		catCmd,
		cdCmd,
		chmodCmd,
		chownCmd,
		cpCmd,
		lsCmd,
		mkdirCmd,
		mvCmd,
		rmCmd,
	}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleCd,
		Cd,
		"bcd",
		Cd,
		common.ParseStatus,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleCat,
		Cat,
		"bcat",
		Cat,
		common.ParseResponse,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleChmod,
		Chmod,
		"",
		nil,
		common.ParseStatus,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleChown,
		Chown,
		"",
		nil,
		common.ParseStatus,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleCp,
		Cp,
		"bcp",
		Cp,
		common.ParseStatus,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleLs,
		Ls,
		"bls",
		Ls,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			err := handler.HandleMaleficError(ctx.Spite)
			if err != nil {
				return "", err
			}
			resp := ctx.Spite.GetLsResponse()
			var fileDetails []string
			for _, file := range resp.GetFiles() {
				fileStr := fmt.Sprintf("%s|%s|%s|%s|%s",
					file.Name,
					strconv.FormatBool(file.IsDir),
					strconv.FormatUint(file.Size, 10),
					strconv.FormatInt(file.ModTime, 10),
					file.Link,
				)
				fileDetails = append(fileDetails, fileStr)
			}
			return strings.Join(fileDetails, ","), nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			msg := content.Spite
			resp := msg.GetLsResponse()
			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				{Title: "name", Width: 25},
				{Title: "size", Width: 10},
				{Title: "mod", Width: 16},
				{Title: "link", Width: 15},
			}, true)
			for _, f := range resp.GetFiles() {
				var size string
				if f.IsDir {
					size = "dir"
				} else {
					size = file.Bytes(f.Size)
				}
				row = table.Row{
					f.Name,
					size,
					time.Unix(f.ModTime, 0).Format("2006-01-02 15:04"),
					f.Link,
				}
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})

	con.RegisterImplantFunc(
		consts.ModuleMkdir,
		Mkdir,
		"bmkdir",
		Mkdir,
		common.ParseStatus,
		nil)

	con.RegisterImplantFunc(
		consts.ModuleMv,
		Mv,
		"bmv",
		Mv,
		common.ParseStatus,
		nil)

	con.RegisterImplantFunc(
		consts.ModulePwd,
		Pwd,
		"bpwd",
		Pwd,
		common.ParseResponse,
		nil,
	)

	con.RegisterImplantFunc(
		consts.ModuleRm,
		Rm,
		"brm",
		Rm,
		common.ParseStatus,
		nil)
}
