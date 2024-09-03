package alias

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	app "github.com/reeflective/console"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	defaultTimeout = 60

	ManifestFileName = "alias.json"

	windowsDefaultHostProc = `c:\windows\system32\notepad.exe`
	linuxDefaultHostProc   = "/bin/bash"
	macosDefaultHostProc   = "/Applications/Safari.app/Contents/MacOS/SafariForWebKitDevelopment"
)

var (
	// alias name -> manifest/command
	loadedAliases = map[string]*loadedAlias{}

	defaultHostProc = map[string]string{
		"windows": windowsDefaultHostProc,
		"linux":   windowsDefaultHostProc,
		"darwin":  macosDefaultHostProc,
	}
)

// Ties the manifest struct to the command struct
type loadedAlias struct {
	Manifest *AliasManifest
	Command  *cobra.Command
}

// AliasFile - An OS/Arch specific file
type AliasFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

// AliasManifest - The manifest for an alias, contains metadata
type AliasManifest struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	CommandName    string `json:"command_name"`
	OriginalAuthor string `json:"original_author"`
	RepoURL        string `json:"repo_url"`
	Help           string `json:"help"`
	LongHelp       string `json:"long_help"`

	Entrypoint   string       `json:"entrypoint"`
	AllowArgs    bool         `json:"allow_args"`
	DefaultArgs  string       `json:"default_args"`
	Files        []*AliasFile `json:"files"`
	IsReflective bool         `json:"is_reflective"`
	IsAssembly   bool         `json:"is_assembly"`

	RootPath   string `json:"-"`
	ArmoryName string `json:"-"`
	ArmoryPK   string `json:"-"`
}

func (ec *AliasManifest) getDefaultProcess(targetOS string) (proc string, err error) {
	proc, ok := defaultHostProc[targetOS]
	if !ok {
		err = fmt.Errorf("no default process for %s target, please specify one", targetOS)
	}
	return
}

func (a *AliasManifest) getFileForTarget(cmdName string, targetOS string, targetArch string) (string, error) {
	filePath := ""
	for _, extFile := range a.Files {
		if targetOS == extFile.OS && targetArch == extFile.Arch {
			filePath = path.Join(assets.GetAliasesDir(), a.CommandName, extFile.Path)
			break
		}
	}
	if filePath == "" {
		err := fmt.Errorf("no alias file found for %s/%s", targetOS, targetArch)
		return "", err
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err = fmt.Errorf("alias file not found: %s", filePath)
		return "", err
	}
	return filePath, nil
}

// AliasesLoadCmd - Locally load a alias into the Sliver shell.
func AliasesLoadCmd(cmd *cobra.Command, con *console.Console) {
	dirPath := cmd.Flags().Arg(0)
	alias, err := LoadAlias(dirPath, con)
	if err != nil {
		console.Log.Errorf("Failed to load alias: %s\n", err)
	} else {
		console.Log.Infof("%s alias has been loaded\n", alias.Name)
	}
	AliasRegisterCommand(alias, con.App.Menu(consts.ImplantMenu).Command, con)
}

// LoadAlias - Load an alias into the Malice-Network shell from a given directory
func LoadAlias(manifestPath string, con *console.Console) (*AliasManifest, error) {
	// retrieve alias manifest
	var err error
	if !strings.HasPrefix(manifestPath, assets.GetAliasesDir()) {
		manifestPath = path.Join(assets.GetAliasesDir(), manifestPath)
	}
	if !files.IsExist(manifestPath) {
		return nil, fmt.Errorf("alias %s maybe not installed", manifestPath)
	}
	// parse it
	if !strings.HasSuffix(manifestPath, "alias.json") {
		manifestPath = path.Join(manifestPath, "alias.json")
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	aliasManifest, err := ParseAliasManifest(data)
	if err != nil {
		return nil, err
	}
	aliasManifest.RootPath = filepath.Dir(manifestPath)
	// for each alias command, add a new app command
	//implantMenu := con.App.Menu(consts.ImplantGroup)
	// do not add if the command already exists
	//if console.CmdExists(aliasManifest.CommandName, implantMenu.Command) {
	//	return nil, fmt.Errorf("'%s' command already exists", aliasManifest.CommandName)
	//}

	return aliasManifest, nil
}

func AliasRegisterCommand(aliasManifest *AliasManifest, cmd *cobra.Command, con *console.Console) {
	helpMsg := fmt.Sprintf("[%s] %s", aliasManifest.Name, aliasManifest.Help)
	longHelpMsg := help.FormatHelpTmpl(aliasManifest.LongHelp)
	longHelpMsg += "\n\n⚠️  If you're having issues passing arguments to the alias please read:\n"
	longHelpMsg += "https://github.com/BishopFox/sliver/wiki/Aliases-&-Extensions#aliases-command-parsing"
	addAliasCmd := &cobra.Command{
		Use:   aliasManifest.CommandName,
		Short: helpMsg,
		Long:  longHelpMsg,
		Run: func(cmd *cobra.Command, args []string) {
			runAliasCommand(cmd, con)
		},
		Args:        cobra.ArbitraryArgs, // 	a.StringList("arguments", "arguments", grumble.Default([]string{}))
		GroupID:     consts.AliasesGroup,
		Annotations: makeAliasPlatformFilters(aliasManifest),
	}

	if aliasManifest.IsAssembly {
		f := pflag.NewFlagSet("assembly", pflag.ContinueOnError)
		//f.StringP("method", "m", "", "Optional method (a method is required for a .NET DLL)")
		//f.StringP("class", "c", "", "Optional class name (required for .NET DLL)")
		//f.StringP("app-domain", "d", "", "AppDomain name to create for .NET assembly. Generated randomly if not set.")
		//f.StringP("arch", "a", "x84", "Assembly target architecture: x86, x64, x84 (x86+x64)")
		//f.BoolP("in-process", "i", false, "Run in the current sliver process")
		//f.StringP("runtime", "r", "", "Runtime to use for running the assembly (only supported when used with --in-process)")
		//f.BoolP("amsi-bypass", "M", false, "Bypass AMSI on Windows (only supported when used with --in-process)")
		//f.BoolP("etw-bypass", "E", false, "Bypass ETW on Windows (only supported when used with --in-process)")
		addAliasCmd.Flags().AddFlagSet(f)
	}

	f := pflag.NewFlagSet(aliasManifest.Name, pflag.ContinueOnError)
	f.StringP("process", "p", "", "Path to process to host the shared object")
	//f.StringP("process-arguments", "A", "", "arguments to pass to the hosting process")
	f.Uint32P("ppid", "P", 0, "parent process ID to use when creating the hosting process (Windows only)")
	f.BoolP("save", "s", false, "Save output to disk")
	f.IntP("timeout", "t", defaultTimeout, "command timeout in seconds")
	addAliasCmd.Flags().AddFlagSet(f)

	loadedAliases[aliasManifest.CommandName] = &loadedAlias{
		Manifest: aliasManifest,
		Command:  addAliasCmd,
	}
	cmd.AddCommand(addAliasCmd)
}

// ParseAliasManifest - Parse an alias manifest
func ParseAliasManifest(data []byte) (*AliasManifest, error) {
	// parse it
	alias := &AliasManifest{}
	err := json.Unmarshal(data, alias)
	if err != nil {
		return nil, err
	}
	if alias.Name == "" {
		return nil, fmt.Errorf("missing alias name in manifest")
	}
	if alias.CommandName == "" {
		return nil, fmt.Errorf("missing command.name in alias manifest")
	}
	if alias.Help == "" {
		return nil, fmt.Errorf("missing command.help in alias manifest")
	}

	for _, aliasFile := range alias.Files {
		if aliasFile.OS == "" {
			return nil, fmt.Errorf("missing command.files.os in alias manifest")
		}
		aliasFile.OS = strings.ToLower(aliasFile.OS)
		if aliasFile.Arch == "" {
			return nil, fmt.Errorf("missing command.files.arch in alias manifest")
		}
		aliasFile.Arch = strings.ToLower(aliasFile.Arch)
		aliasFile.Path = utils.ResolvePath(aliasFile.Path)
		if aliasFile.Path == "" || aliasFile.Path == "/" {
			return nil, fmt.Errorf("missing command.files.path in alias manifest")
		}
	}

	return alias, nil
}

func runAliasCommand(cmd *cobra.Command, con *console.Console) {
	session := con.GetInteractive()
	if session == nil {
		return
	}
	var goos string
	var goarch string
	goos = session.Os.Name
	goarch = session.Os.Arch

	loadedAlias, ok := loadedAliases[cmd.Name()]
	if !ok {
		console.Log.Errorf("No alias found for `%s` command\n", cmd.Name())
		return
	}
	aliasManifest := loadedAlias.Manifest
	binPath, err := aliasManifest.getFileForTarget(cmd.Name(), goos, goarch)
	if err != nil {
		console.Log.Errorf("Fail to find alias file: %s\n", err)
		return
	}
	args := cmd.Flags().Args()
	var extArgs string
	if len(aliasManifest.DefaultArgs) != 0 && len(args) == 0 {
		extArgs = aliasManifest.DefaultArgs
	} else {
		extArgs = strings.Join(args, " ")
	}

	extArgs = strings.TrimSpace(extArgs)
	//entryPoint := aliasManifest.Entrypoint
	//processArgsStr := ctx.Flags.String("process-arguments")
	// Special case for payloa ds with pass to Donut (.NET assemblies and sideloaded payloads):
	// The Donut loader has a hard limit of 256 characters for the command line arguments, so
	// we're alerting the user that the arguments will be truncated.
	if len(extArgs) > 256 && (aliasManifest.IsAssembly || !aliasManifest.IsReflective) {
		msgStr := ""
		// The --in-process flag only exists for .NET assemblies (aliasManifest.IsAssembly == true).
		// Groupping the two conditions together could crash the client since ctx.Flags.Type panics
		// if the flag is not registered.
		if aliasManifest.IsAssembly {
			inProcess, _ := cmd.Flags().GetBool("in-process")
			runtime, _ := cmd.Flags().GetString("runtime")
			amsiBypass, _ := cmd.Flags().GetBool("amsi-bypass")
			etwBypass, _ := cmd.Flags().GetBool("etw-bypass")
			if !inProcess {
				msgStr = " Arguments are limited to 256 characters when using the default fork/exec model for .NET assemblies.\nConsider using the --in-process flag to execute .NET assemblies in-process and work around this limitation.\n"
			}
			if !inProcess && (runtime != "" || etwBypass || amsiBypass) {
				console.Log.Errorf("The --runtime, --etw-bypass, and --amsi-bypass flags can only be used with the --in-process flag\n")
				return
			}
		} else if !aliasManifest.IsReflective {
			msgStr = " Arguments are limited to 256 characters when using the default fork/exec model for non-reflective PE payloads.\n"
		}
		console.Log.Warn(msgStr)
		//confirm := false
		//prompt := &survey.Confirm{Message: "Do you want to continue?"}
		//survey.AskOne(prompt, &confirm, nil)
		//if !confirm {
		//	return
		//}
	}
	//processArgs := strings.Split(processArgsStr, " ")
	processName, _ := cmd.Flags().GetString("process")
	if processName == "" {
		processName, err = aliasManifest.getDefaultProcess(goos)
		if err != nil {
			console.Log.Errorf("%s\n", err)
			return
		}
	}
	//isDLL := false
	//if strings.ToLower(filepath.Ext(binPath)) == ".dll" {
	//	isDLL = true
	//}
	binData, err := os.ReadFile(binPath)
	if err != nil {
		console.Log.Errorf("%s\n", err)
		return
	}

	if aliasManifest.IsAssembly {
		executeAssemblyResp, err := con.Rpc.ExecuteAssembly(con.ActiveTarget.Context(), &implantpb.ExecuteBinary{
			Name:   loadedAlias.Command.Name(),
			Bin:    binData,
			Type:   consts.ModuleExecuteAssembly,
			Params: args,
		})
		if err != nil {
			console.Log.Errorf("%s\n", err)
			return
		}

		con.AddCallback(executeAssemblyResp.TaskId, func(msg proto.Message) {
			resp := msg.(*implantpb.Spite).GetAssemblyResponse()
			sid := con.GetInteractive().SessionId
			if resp.Status == 0 {
				con.SessionLog(sid).Infof("%s output:\n%s", loadedAlias.Command.Name, string(resp.Data))
			} else {
				con.SessionLog(sid).Errorf("%s %s ", cmd.Name(), resp.Err)
			}
		})

		//} else if aliasManifest.IsReflective {
		//
		//	// Spawn DLL
		//	ctrl := make(chan bool)
		//	msg := fmt.Sprintf("Executing %s %s ...", ctx.Command.Name, extArgs)
		//	con.SpinUntil(msg, ctrl)
		//	spawnDllResp, err := con.Rpc.SpawnDll(context.Background(), &sliverpb.InvokeSpawnDllReq{
		//		Request:     con.ActiveTarget.Request(ctx),
		//		Args:        strings.Trim(extArgs, " "),
		//		Data:        binData,
		//		ProcessName: processName,
		//		EntryPoint:  aliasManifest.Entrypoint,
		//		Kill:        true,
		//		ProcessArgs: processArgs,
		//		PPid:        uint32(ctx.Flags.Uint("ppid")),
		//	})
		//	ctrl <- true
		//	<-ctrl
		//	if err != nil {
		//		console.Log.Errorf("%s\n", err)
		//		return
		//	}
		//
		//	if spawnDllResp.Response != nil && spawnDllResp.Response.Async {
		//		con.AddBeaconCallback(spawnDllResp.Response.TaskID, func(task *clientpb.BeaconTask) {
		//			err = proto.Unmarshal(task.Response, spawnDllResp)
		//			if err != nil {
		//				console.Log.Errorf("Failed to decode call ext response %s\n", err)
		//				return
		//			}
		//			PrintSpawnDLLOutput(ctx.Command.Name, spawnDllResp, outFilePath, con)
		//		})
		//		con.PrintAsyncResponse(spawnDllResp.Response)
		//	} else {
		//		PrintSpawnDLLOutput(ctx.Command.Name, spawnDllResp, outFilePath, con)
		//	}
		//
		//} else {
		//
		//	// Sideload
		//	ctrl := make(chan bool)
		//	msg := fmt.Sprintf("Executing %s %s ...", ctx.Command.Name, extArgs)
		//	con.SpinUntil(msg, ctrl)
		//	sideloadResp, err := con.Rpc.Sideload(context.Background(), &sliverpb.SideloadReq{
		//		Request:     con.ActiveTarget.Request(ctx),
		//		Args:        extArgs,
		//		Data:        binData,
		//		EntryPoint:  entryPoint,
		//		ProcessName: processName,
		//		Kill:        true,
		//		IsDLL:       isDLL,
		//		ProcessArgs: processArgs,
		//		PPid:        uint32(ctx.Flags.Uint("ppid")),
		//	})
		//	ctrl <- true
		//	<-ctrl
		//	if err != nil {
		//		console.Log.Errorf("%s\n", err)
		//		return
		//	}
		//
		//	if sideloadResp.Response != nil && sideloadResp.Response.Async {
		//		con.AddBeaconCallback(sideloadResp.Response.TaskID, func(task *clientpb.BeaconTask) {
		//			err = proto.Unmarshal(task.Response, sideloadResp)
		//			if err != nil {
		//				console.Log.Errorf("Failed to decode call ext response %s\n", err)
		//				return
		//			}
		//			PrintSideloadOutput(ctx.Command.Name, sideloadResp, outFilePath, con)
		//		})
		//		con.PrintAsyncResponse(sideloadResp.Response)
		//	} else {
		//		PrintSideloadOutput(ctx.Command.Name, sideloadResp, outFilePath, con)
		//	}
	}
}

// PrintSpawnDLLOutput - Prints the output of a spawn dll command
//func PrintSpawnDLLOutput(cmdName string, spawnDllResp *sliverpb.SpawnDll, outFilePath *os.File, con *console.Console) {
//	console.Log.Infof("%s output:\n%s", cmdName, spawnDllResp.GetResult())
//	if outFilePath != nil {
//		outFilePath.Write([]byte(spawnDllResp.GetResult()))
//		console.Log.Infof("Output saved to %s\n", outFilePath.Name())
//	}
//}
//
//// PrintSideloadOutput - Prints the output of a sideload command
//func PrintSideloadOutput(cmdName string, sideloadResp *sliverpb.Sideload, outFilePath *os.File, con *console.Console) {
//	console.Log.Infof("%s output:\n%s", cmdName, sideloadResp.GetResult())
//	if outFilePath != nil {
//		outFilePath.Write([]byte(sideloadResp.GetResult()))
//		console.Log.Infof("Output saved to %s\n", outFilePath.Name())
//	}
//}

//// PrintAssemblyOutput - Prints the output of an execute-assembly command
//func PrintAssemblyOutput(cmdName string, resp *clientpb.Task, outFilePath *os.File, con *console.Console) {
//	console.Log.Infof("%s output:\n%s", cmdName, string(resp.Need))
//	if outFilePath != nil {
//		outFilePath.Write(resp.Need)
//		console.Log.Infof("Output saved to %s\n", outFilePath.Name())
//	}
//}

func makeAliasPlatformFilters(alias *AliasManifest) map[string]string {
	filtersOS := make(map[string]bool)
	filtersArch := make(map[string]bool)

	var all []string

	// Only add filters for architectures when there OS matters.
	for _, file := range alias.Files {
		filtersOS[file.OS] = true

		if filtersOS[file.OS] {
			filtersArch[file.Arch] = true
		}
	}

	for os, enabled := range filtersOS {
		if enabled {
			all = append(all, os)
		}
	}

	for arch, enabled := range filtersArch {
		if enabled {
			all = append(all, arch)
		}
	}

	if len(all) == 0 {
		return map[string]string{}
	}

	return map[string]string{
		app.CommandFilterKey: strings.Join(all, ","),
	}
}
