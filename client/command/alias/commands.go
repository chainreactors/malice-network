package alias

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func Commands(con *repl.Console) []*cobra.Command {
	aliasCmd := &cobra.Command{
		Use:   consts.CommandAlias,
		Short: "manage aliases",
		Long: `
Macros are using the sideload or spawndll commands under the hood, depending on the use case. 

For Linux and Mac OS, the sideload command will be used. On Windows, it will depend on whether the macro file is a reflective DLL or not. 

Load a macro: 
~~~
load /tmp/chrome-dump 
~~~

Sliver macros have the following structure (example for the chrome-dump macro): 

chrome-dump 
* chrome-dump.dll 
* chrome-dump.so 
* manifest.json

It is a directory containing any number of files, with a mandatory manifest.json, that has the following structure: 

~~~
{ 
	"macroName":"chrome-dump", // name of the macro, can be anything
	"macroCommands":[ 
		{ 
			"name":"chrome-dump", // name of the command available in the sliver client (no space)
			"entrypoint":"ChromeDump", // entrypoint of the shared library to execute
			"help":"Dump Google Chrome cookies", // short help message
			"allowArgs":false, // make it true if the commands require arguments
			"defaultArgs": "test", // if you need to pass a default argument
			"extFiles":[ // list of files, groupped per target OS
				{ 
					"os":"windows", // Target OS for the following files. Values can be "windows", "linux" or "darwin" 
					"files":{ 
						"x64":"chrome-dump.dll", 
						"x86":"chrome-dump.x86.dll" // only x86 and x64 arch are supported, path is relative to the macro directory
					} 
				}, 
				{
					"os":"linux", 
					"files":{
						"x64":"chrome-dump.so" 
					} 
				}, 
				{
					"os":"darwin", 
					"files":{ 
						"x64":"chrome-dump.dylib"
						} 
					} 
				], 
			"isReflective":false // only set to true when using a reflective DLL
		} 
	] 
} 
~~~

Each command will have the --process flag defined, which allows you to specify the process to inject into. The following default values are set:
	
	- Windows: c:\windows\system32\notepad.exe 
	- Linux: /bin/bash 
	- Mac OS X: /Applications/Safari.app/Contents/MacOS/SafariForWebKitDevelopment
`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			return
		},
	}

	aliasListCmd := &cobra.Command{
		Use:   consts.CommandAliasList,
		Short: "List all aliases",
		Long:  "See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions",
		Run: func(cmd *cobra.Command, args []string) {
			AliasesCmd(cmd, con)
			return
		},
	}

	aliasLoadCmd := &cobra.Command{
		Use:   consts.CommandAliasLoad + " [alias]",
		Short: "Load a command alias",
		Long:  "See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			AliasesLoadCmd(cmd, con)
			return
		},
		Example: `
~~~
// Load a command alias
alias load /tmp/chrome-dump
~~~`,
	}
	common.BindArgCompletions(
		aliasLoadCmd,
		nil,
		carapace.ActionFiles().Usage("local path where the downloaded file will be saved (optional)"),
	)

	aliasInstallCmd := &cobra.Command{
		Use:   consts.CommandAliasInstall + " [alias_file]",
		Short: "Install a command alias",
		Long:  "See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			AliasesInstallCmd(cmd, con)
			return
		},
		Example: `
~~~
// Install a command alias
alias install ./rubeus.exe
~~~`,
	}

	common.BindArgCompletions(aliasInstallCmd,
		nil,
		carapace.ActionFiles().Usage("local path where the downloaded file will be saved (optional)"),
	)

	aliasRemoveCmd := &cobra.Command{
		Use:   consts.CommandAliasRemove + " [alias]",
		Short: "Remove an alias",
		Long:  "See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			AliasesRemoveCmd(cmd, con)
			return
		},
		Example: `
~~~
// Remove an alias
alias remove rubeus
~~~`,
	}

	common.BindArgCompletions(
		aliasRemoveCmd,
		nil,
		AliasCompleter())

	aliasCmd.AddCommand(aliasListCmd, aliasLoadCmd, aliasInstallCmd, aliasRemoveCmd)
	return []*cobra.Command{aliasCmd}

}

func Register(con *repl.Console) {
	for name, aliasPkg := range loadedAliases {
		intermediate.RegisterInternalFunc(intermediate.ArmoryPackage, name, aliasPkg.Func, repl.WrapClientCallback(common.ParseAssembly))
	}
}
