package filesystem

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
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

func Commands(con *console.Console) []*cobra.Command {

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

	carapace.Gen(catCmd).PositionalCompletion(
		carapace.ActionValues().Usage("cat file name"),
	)

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

	carapace.Gen(cdCmd).PositionalCompletion(
		carapace.ActionValues().Usage("cd path"),
	)

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

	carapace.Gen(chmodCmd).PositionalCompletion(
		carapace.ActionValues().Usage("chmod file mode"),
		carapace.ActionValues().Usage("chmod file path"),
	)

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

	carapace.Gen(chownCmd).PositionalCompletion(
		carapace.ActionValues().Usage("chown user"),
		carapace.ActionValues().Usage("chown file path"),
	)

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

	carapace.Gen(cpCmd).PositionalCompletion(
		carapace.ActionValues().Usage("source file"),
		carapace.ActionValues().Usage("target file"),
	)

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

	carapace.Gen(lsCmd).PositionalCompletion(
		carapace.ActionValues().Usage("ls path"),
	)

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

	carapace.Gen(mkdirCmd).PositionalCompletion(
		carapace.ActionValues().Usage("mkdir path"),
	)

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

	carapace.Gen(mvCmd).PositionalCompletion(
		carapace.ActionValues().Usage("source file"),
		carapace.ActionValues().Usage("target file"),
	)

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

	carapace.Gen(rmCmd).PositionalCompletion(
		carapace.ActionValues().Usage("rm file name"),
	)

	con.RegisterInternalFunc(
		"bcd",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, path string) (*clientpb.Task, error) {
			return Cd(rpc, sess, path)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"cat",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, fileName string) (*clientpb.Task, error) {
			return Cat(rpc, sess, fileName)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			err := handler.HandleMaleficError(ctx.GetSpite())
			if err != nil {
				return "", err
			}
			resp := ctx.GetSpite().GetResponse()
			return resp.GetOutput(), nil
		})

	con.RegisterInternalFunc(
		"chmod",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, path, mode string) (*clientpb.Task, error) {
			return Chmod(rpc, sess, path, mode)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"chown",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, path, uid string, gid string, recursive bool) (*clientpb.Task, error) {
			return Chown(rpc, sess, path, uid, gid, recursive)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"bcp",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, src, dst string) (*clientpb.Task, error) {
			return Cp(rpc, sess, src, dst)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"bls",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, path string) (*clientpb.Task, error) {
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

	con.RegisterInternalFunc(
		"bmkdir",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, path string) (*clientpb.Task, error) {
			return Mkdir(rpc, sess, path)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"bmv",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, src, dst string) (*clientpb.Task, error) {
			return Mv(rpc, sess, src, dst)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

	con.RegisterInternalFunc(
		"bpwd",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session) (*clientpb.Task, error) {
			return Pwd(rpc, sess)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			err := handler.HandleMaleficError(ctx.Spite)
			if err != nil {
				return "", err
			}
			resp := ctx.Spite.GetResponse()
			return resp.GetOutput(), nil
		})

	con.RegisterInternalFunc(
		"brm",
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, fileName string) (*clientpb.Task, error) {
			return Rm(rpc, sess, fileName)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseStatus(ctx.Spite)
		})

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
