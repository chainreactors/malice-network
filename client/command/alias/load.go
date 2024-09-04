package alias

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/files"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/core/intermediate/builtin"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
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
	err = RegisterAlias(alias, con.App.Menu(consts.ImplantMenu).Command, con)
	if err != nil {
		console.Log.Errorf(err.Error())
		return
	}
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
	if !strings.HasSuffix(manifestPath, ManifestFileName) {
		manifestPath = path.Join(manifestPath, ManifestFileName)
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

func RegisterAlias(aliasManifest *AliasManifest, cmd *cobra.Command, con *console.Console) error {
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
	f.BoolP("block_dll", "B", false, "block non-microsoft dll")
	f.Uint32P("ppid", "P", 0, "parent process ID to use when creating the hosting process (Windows only)")
	f.StringP("argue", "a", "", "argue")
	f.BoolP("save", "s", false, "Save output to disk")
	f.IntP("timeout", "t", defaultTimeout, "command timeout in seconds")
	addAliasCmd.Flags().AddFlagSet(f)

	loadedAliases[aliasManifest.CommandName] = &loadedAlias{
		Manifest: aliasManifest,
		Command:  addAliasCmd,
	}
	cmd.AddCommand(addAliasCmd)
	err := con.RegisterInternalFunc(
		aliasManifest.CommandName,
		func(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, args string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
			return ExecuteAlias(rpc, sess, aliasManifest.CommandName, args, sac)
		},
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return builtin.ParseAssembly(ctx.Spite)
		})
	if err != nil {
		return err
	}
	return nil
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
	sid := session.SessionId
	loadedAlias, ok := loadedAliases[cmd.Name()]
	if !ok {
		console.Log.Errorf("No alias found for `%s` command\n", cmd.Name())
		return
	}
	aliasManifest := loadedAlias.Manifest
	args := cmd.Flags().Args()
	var extArgs string
	if len(aliasManifest.DefaultArgs) != 0 && len(args) == 0 {
		extArgs = aliasManifest.DefaultArgs
	} else {
		extArgs = strings.Join(args, " ")
	}

	extArgs = strings.TrimSpace(extArgs)
	var task *clientpb.Task
	var err error
	isInline, _ := cmd.Flags().GetBool("in-process")
	if isInline {
		task, err = ExecuteAlias(con.Rpc, session, cmd.Name(), extArgs, nil)
	} else {
		processName, _ := cmd.Flags().GetString("process")
		if processName == "" {
			processName, err = aliasManifest.getDefaultProcess(con.GetInteractive().Os.Name)
			if err != nil {
				console.Log.Errorf("%s\n", err)
				return
			}
		}
		ppid, _ := cmd.Flags().GetUint32("ppid")
		block_dll, _ := cmd.Flags().GetBool("block-dll")
		argue, _ := cmd.Flags().GetString("argue")
		sac, _ := builtin.NewSacrificeProcessMessage(processName, int64(ppid), block_dll, argue, extArgs)
		task, err = ExecuteAlias(con.Rpc, session, cmd.Name(), extArgs, sac)
	}

	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetAssemblyResponse()
		if resp.Status == 0 {
			con.SessionLog(sid).Infof("%s output:\n%s", loadedAlias.Command.Name(), string(resp.Data))
		} else {
			con.SessionLog(sid).Errorf("%s %s ", cmd.Name(), resp.Err)
		}
	})
}

func ExecuteAlias(rpc clientrpc.MaliceRPCClient, sess *clientpb.Session, aliasName string, args string, sac *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	loadedAlias, ok := loadedAliases[aliasName]
	if !ok {
		return nil, fmt.Errorf("No alias found for `%s` command\n", aliasName)
	}
	aliasManifest := loadedAlias.Manifest
	binPath, err := aliasManifest.getFileForTarget(aliasName, sess.Os.Name, sess.Os.Arch)
	if err != nil {
		return nil, fmt.Errorf("Fail to find alias file: %w\n", err)
	}

	binData, err := os.ReadFile(binPath)
	if err != nil {
		return nil, err
	}
	var task *clientpb.Task
	if aliasManifest.IsAssembly {
		task, err = rpc.ExecuteAssembly(console.Context(sess), &implantpb.ExecuteBinary{
			Name: loadedAlias.Command.Name(),
			Bin:  binData,
			Type: consts.ModuleExecuteAssembly,
		})
	} else {
		task, err = rpc.ExecuteDLL(console.Context(sess), &implantpb.ExecuteBinary{
			Name:       loadedAlias.Command.Name(),
			Bin:        binData,
			EntryPoint: aliasManifest.Entrypoint,
			Type:       consts.ModuleExecuteDll,
			Sacrifice:  sac,
		})
	}
	if err != nil {
		return nil, err
	}
	return task, nil
}

func makeAliasPlatformFilters(alias *AliasManifest) map[string]string {
	all := make(map[string]string)

	// Only add filters for architectures when there OS matters.
	var arch []string
	for _, file := range alias.Files {
		all["os"] = file.OS
		arch = append(arch, file.Arch)
	}
	all["arch"] = strings.Join(arch, ",")

	if alias.IsAssembly {
		all["depend"] = consts.ModuleExecuteAssembly
	} else if alias.IsReflective {
		all["depend"] = consts.ModuleExecuteDll
	}
	return all
}
