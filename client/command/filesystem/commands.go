package filesystem

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/handler"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strconv"
	"strings"
)

func Commands(con *repl.Console) []*cobra.Command {

	pwdCmd := &cobra.Command{
		Use:   consts.ModulePwd,
		Short: "Print working directory",
		Long:  help.GetHelpFor(consts.ModulePwd),
		Run: func(cmd *cobra.Command, args []string) {
			PwdCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleCat,
		},
	}

	catCmd := &cobra.Command{
		Use:   consts.ModuleCat,
		Short: "Print file content",
		Long:  help.GetHelpFor(consts.ModuleCat),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			CatCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleCat,
		},
	}

	common.BindArgCompletions(catCmd, nil,
		carapace.ActionValues().Usage("cat file name"))

	cdCmd := &cobra.Command{
		Use:   consts.ModuleCd,
		Short: "Change directory",
		Long:  help.GetHelpFor(consts.ModuleCd),
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
		Use:   consts.ModuleChmod,
		Short: "Change file mode",
		Long:  help.GetHelpFor(consts.ModuleChmod),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ChmodCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"os":     "linux,mac",
			"depend": consts.ModuleChmod,
		},
	}

	common.BindArgCompletions(chmodCmd, nil,
		carapace.ActionValues().Usage("chmod file mode"),
		carapace.ActionValues().Usage("chmod file path"))

	chownCmd := &cobra.Command{
		Use:   consts.ModuleChown,
		Short: "Change file owner",
		Long:  help.GetHelpFor(consts.ModuleChown),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ChownCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"os":     "linux,mac",
			"depend": consts.ModuleChown,
		},
	}

	common.BindArgCompletions(chownCmd, nil,
		carapace.ActionValues().Usage("chown user"),
		carapace.ActionValues().Usage("chown file path"))

	common.BindFlag(chownCmd, func(f *pflag.FlagSet) {
		f.BoolP("recursive", "r", false, "recursive")
		f.StringP("gid", "g", "", "Group id")
	})

	cpCmd := &cobra.Command{
		Use:   consts.ModuleCp,
		Short: "Copy file",
		Long:  help.GetHelpFor(consts.ModuleCp),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			CpCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleCp,
		},
	}

	common.BindArgCompletions(cpCmd, nil,
		carapace.ActionValues().Usage("source file"),
		carapace.ActionValues().Usage("target file"))

	lsCmd := &cobra.Command{
		Use:   consts.ModuleLs,
		Short: "List directory",
		Long:  help.GetHelpFor(consts.ModuleLs),
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			LsCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleLs,
		},
	}

	common.BindArgCompletions(lsCmd, nil,
		carapace.ActionValues().Usage("ls path"))

	mkdirCmd := &cobra.Command{
		Use:   consts.ModuleMkdir,
		Short: "Make directory",
		Long:  help.GetHelpFor(consts.ModuleMkdir),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			MkdirCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleMkdir,
		},
	}

	common.BindArgCompletions(mkdirCmd, nil,
		carapace.ActionValues().Usage("mkdir path"))

	mvCmd := &cobra.Command{
		Use:   consts.ModuleMv,
		Short: "Move file",
		Long:  help.GetHelpFor(consts.ModuleMv),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			MvCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleMv,
		},
	}

	common.BindArgCompletions(mvCmd, nil,
		carapace.ActionValues().Usage("source file"),
		carapace.ActionValues().Usage("target file"))

	rmCmd := &cobra.Command{
		Use:   consts.ModuleRm,
		Short: "Remove file",
		Long:  help.GetHelpFor(consts.ModuleRm),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RmCmd(cmd, con)
			return
		},
		Annotations: map[string]string{
			"depend": consts.ModuleRm,
		},
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
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string) (*clientpb.Task, error) {
			return Cd(rpc, sess, path)
		},
		common.ParseStatus)

	con.RegisterImplantFunc(
		consts.ModuleCat,
		Cat,
		"bcat",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, fileName string) (*clientpb.Task, error) {
			return Cat(rpc, sess, fileName)
		},
		common.ParseResponse)

	con.RegisterImplantFunc(
		consts.ModuleChmod,
		Chmod,
		"bchmod",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path, mode string) (*clientpb.Task, error) {
			return Chmod(rpc, sess, path, mode)
		},
		common.ParseStatus)

	con.RegisterImplantFunc(
		consts.ModuleChown,
		Chown,
		"bchown",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path, uid string, gid string, recursive bool) (*clientpb.Task, error) {
			return Chown(rpc, sess, path, uid, gid, recursive)
		},
		common.ParseStatus)

	con.RegisterImplantFunc(
		consts.ModuleCp,
		Cp,
		"bcp",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, src, dst string) (*clientpb.Task, error) {
			return Cp(rpc, sess, src, dst)
		},
		common.ParseStatus)

	con.RegisterImplantFunc(
		consts.ModuleLs,
		Ls,
		"bls",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string) (*clientpb.Task, error) {
			return Ls(rpc, sess, path)
		},
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
		})

	con.RegisterImplantFunc(
		consts.ModuleMkdir,
		Mkdir,
		"bmkdir",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, path string) (*clientpb.Task, error) {
			return Mkdir(rpc, sess, path)
		},
		common.ParseStatus)

	con.RegisterImplantFunc(
		consts.ModuleMv,
		Mv,
		"bmv",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, src, dst string) (*clientpb.Task, error) {
			return Mv(rpc, sess, src, dst)
		},
		common.ParseStatus)

	con.RegisterImplantFunc(
		consts.ModulePwd,
		Pwd,
		"bpwd",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
			return Pwd(rpc, sess)
		},
		common.ParseResponse,
	)

	con.RegisterImplantFunc(
		consts.ModuleRm,
		Rm,
		"brm",
		func(rpc clientrpc.MaliceRPCClient, sess *core.Session, fileName string) (*clientpb.Task, error) {
			return Rm(rpc, sess, fileName)
		},
		common.ParseStatus)
}
