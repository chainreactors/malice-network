package filesystem

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	pwdCmd := &cobra.Command{
		Use:   consts.ModulePwd,
		Short: "Print working directory",
		Long:  "print working directory in implant",
		RunE: func(cmd *cobra.Command, args []string) error {
			return PwdCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return CatCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return CdCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return ChmodCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return ChownCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return CpCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return LsCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return MkdirCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return MvCmd(cmd, con)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return RmCmd(cmd, con)
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

	con.AddCommandFuncHelper(
		consts.ModuleCd,
		consts.ModuleCd,
		consts.ModuleCd+`(active(),"path")`,
		[]string{
			"session: special session",
			"path: path to change directory",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleCat,
		Cat,
		"bcat",
		Cat,
		common.ParseResponse,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleCat,
		consts.ModuleCat,
		consts.ModuleCat+`(active(),"file.txt")`,
		[]string{
			"session: special session",
			"fileName: file to print",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleChmod,
		Chmod,
		"",
		nil,
		common.ParseStatus,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleChmod,
		consts.ModuleChmod,
		consts.ModuleChmod+`(active(),"file.txt","644")`,
		[]string{
			"session: special session",
			"path: file to change mode",
			"mode: mode to change",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleChown,
		Chown,
		"",
		nil,
		common.ParseStatus,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleChown,
		consts.ModuleChown,
		consts.ModuleChown+`(active(),"file.txt","username","groupname",true)`,
		[]string{
			"session: special session",
			"path: file to change owner",
			"uid: user to change",
			"gid: group to change",
			"recursive: recursive",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleCp,
		Cp,
		"bcp",
		Cp,
		common.ParseStatus,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleCp,
		consts.ModuleCp,
		consts.ModuleCp+`(active(),"source","target")`,
		[]string{
			"session: special session",
			"originPath: origin path",
			"targetPath: target path",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleMkdir,
		Mkdir,
		"bmkdir",
		Mkdir,
		common.ParseStatus,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleMkdir,
		consts.ModuleMkdir,
		consts.ModuleMkdir+`(active(),"/tmp")`,
		[]string{
			"session: special session",
			"path: dir",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleMv,
		Mv,
		"bmv",
		Mv,
		common.ParseStatus,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleMv,
		consts.ModuleMv,
		consts.ModuleMv+`(active(),"/tmp/file1.txt","/tmp/file2.txt")`,
		[]string{
			"session: special session",
			"sourcePath: source path",
			"targetPath: target path",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModulePwd,
		Pwd,
		"bpwd",
		Pwd,
		common.ParseResponse,
		nil,
	)

	con.AddCommandFuncHelper(
		consts.ModulePwd,
		consts.ModulePwd,
		consts.ModulePwd+"(active())",
		[]string{
			"session: special session",
		},
		[]string{"task"})

	con.RegisterImplantFunc(
		consts.ModuleRm,
		Rm,
		"brm",
		Rm,
		common.ParseStatus,
		nil)

	con.AddCommandFuncHelper(
		consts.ModuleRm,
		consts.ModuleRm,
		consts.ModulePwd+`(active(),"/tmp/file.txt")`,
		[]string{
			"session: special session",
			"fileName: file to remove",
		},
		[]string{"task"})

	RegisterLsFunc(con)
}
