package extension

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/tui"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	defaultTimeout = 60

	// ManifestFileName - Extension manifest file name
	ManifestFileName = "extension.json"
)

var (
	ErrExtensionDependModuleNotFound = errors.New("extension depends on module not found")
	DependOnMap                      = map[string]string{
		"coff-loader": consts.ModuleExecuteBof,
	}
)

var loadedExtensions = map[string]*ExtCommand{}
var loadedManifests = map[string]*ExtensionManifest{}

type ExtensionManifest_ struct {
	Name            string               `json:"name"`
	CommandName     string               `json:"command_name"`
	Version         string               `json:"version"`
	ExtensionAuthor string               `json:"extension_author"`
	OriginalAuthor  string               `json:"original_author"`
	RepoURL         string               `json:"repo_url"`
	Help            string               `json:"help"`
	LongHelp        string               `json:"long_help"`
	Files           []*extensionFile     `json:"files"`
	Arguments       []*extensionArgument `json:"arguments"`
	Entrypoint      string               `json:"entrypoint"`
	DependsOn       string               `json:"depends_on"`
	Init            string               `json:"init"`

	RootPath string `json:"-"`
}

type ExtensionManifest struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	ExtensionAuthor string `json:"extension_author"`
	OriginalAuthor  string `json:"original_author"`
	RepoURL         string `json:"repo_url"`

	ExtCommand []*ExtCommand `json:"commands"`

	RootPath   string `json:"-"`
	ArmoryName string `json:"-"`
	ArmoryPK   string `json:"-"`
}

type ExtCommand struct {
	CommandName string               `json:"command_name"`
	Help        string               `json:"help"`
	LongHelp    string               `json:"long_help"`
	Files       []*extensionFile     `json:"files"`
	Arguments   []*extensionArgument `json:"arguments"`
	Entrypoint  string               `json:"entrypoint"`
	DependsOn   string               `json:"depends_on"`
	Init        string               `json:"init"`

	Manifest *ExtensionManifest
}

type extensionFile struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
	Path string `json:"path"`
}

type extensionArgument struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Desc     string `json:"desc"`
	Optional bool   `json:"optional"`
}

func (e *ExtCommand) getFileForTarget(targetOS string, targetArch string) (string, error) {
	filePath := ""
	for _, extFile := range e.Files {
		if targetOS == extFile.OS && targetArch == extFile.Arch {
			filePath = filepath.Join(assets.GetExtensionsDir(), e.CommandName, extFile.Path)
			break
		}
	}
	if filePath == "" {
		err := fmt.Errorf("no extension file found for %s/%s", targetOS, targetArch)
		return "", err
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err = fmt.Errorf("extension file not found: %s", filePath)
		return "", err
	}
	return filePath, nil
}

// ExtensionLoadCmd - Load extension command
func ExtensionLoadCmd(ctx *grumble.Context, con *console.Console) {
	dirPath := ctx.Args.String("dir-path")
	manifest, err := LoadExtensionManifest(filepath.Join(assets.GetExtensionsDir(), dirPath, ManifestFileName))
	if err != nil {
		return
	}
	// do not add if the command already exists
	for _, extCmd := range manifest.ExtCommand {
		if console.CmdExists(extCmd.CommandName, con.App) {
			console.Log.Errorf("%s command already exists\n", extCmd.CommandName)
			confirmModel := tui.NewConfirm(fmt.Sprintf("%s command already exists. Overwrite?", extCmd.CommandName))
			newConfirm := tui.NewModel(confirmModel, nil, false, true)
			err = newConfirm.Run()
			if err != nil {
				console.Log.Errorf("Error running confirm model: %s\n", err)
				return
			}
			if !confirmModel.Confirmed {
				return
			}
		}
		ExtensionRegisterCommand(extCmd, con)
		console.Log.Infof("Added %s command: %s\n", extCmd.CommandName, extCmd.Help)
	}
}

// LoadExtensionManifest - Parse extension files
func LoadExtensionManifest(manifestPath string) (*ExtensionManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	manifest, err := ParseExtensionManifest(data)
	if err != nil {
		return nil, err
	}
	manifest.RootPath = filepath.Dir(manifestPath)
	for _, extManifest := range manifest.ExtCommand {
		loadedExtensions[extManifest.CommandName] = extManifest
	}
	loadedManifests[manifest.Name] = manifest
	return manifest, nil
}

// ParseExtensionManifest - Parse extension manifest from buffer
func ParseExtensionManifest(data []byte) (*ExtensionManifest, error) {
	extManifest := &ExtensionManifest{}
	err := json.Unmarshal(data, &extManifest)
	if err != nil || len(extManifest.ExtCommand) == 0 {
		if err != nil {
			console.Log.Errorf("extension load error: %s\n", err)
		}
		oldmanifest := &ExtensionManifest_{}
		err = json.Unmarshal(data, &oldmanifest)
		if err != nil {
			return nil, err
		}
		extManifest = convertOldManifest(oldmanifest)
	}
	for i := range extManifest.ExtCommand {
		command := extManifest.ExtCommand[i]
		command.Manifest = extManifest
	}
	return extManifest, validManifest(extManifest)
}

// ExtensionRegisterCommand - Register a new extension command
func ExtensionRegisterCommand(extCmd *ExtCommand, con *console.Console) {
	loadedExtensions[extCmd.CommandName] = extCmd
	helpMsg := extCmd.Help
	extensionCmd := &grumble.Command{
		Name: extCmd.CommandName,
		Help: helpMsg,
		//LongHelp: help.FormatHelpTmpl(extCmd.LongHelp),
		Run: func(extCtx *grumble.Context) error {
			runExtensionCmd(extCtx, con)
			return nil
		},
		Flags: func(f *grumble.Flags) {
			// f.Bool("s", "save", false, "Save output to disk")
			f.Int("t", "timeout", defaultTimeout, "command timeout in seconds")
		},
		Args: func(a *grumble.Args) {
			if 0 < len(extCmd.Arguments) {
				// BOF specific
				for _, arg := range extCmd.Arguments {
					var (
						argFunc      func(string, string, ...grumble.ArgOption)
						defaultValue grumble.ArgOption
					)
					switch arg.Type {
					case "int", "integer", "short":
						argFunc = a.Int
						defaultValue = grumble.Default(0)
					case "string", "wstring", "file":
						argFunc = a.String
						defaultValue = grumble.Default("")
					default:
						console.Log.Errorf("Invalid argument type: %s\n", arg.Type)
						return
					}
					if arg.Optional {
						argFunc(arg.Name, arg.Desc, defaultValue)
					} else {
						argFunc(arg.Name, arg.Desc)
					}
				}
			} else {
				a.StringList("arguments", "arguments", grumble.Default([]string{}))
			}
		},
		HelpGroup: consts.ExtensionGroup,
	}

	con.AddExtensionCommand(extensionCmd)
}

func loadExtension(goos string, goarch string, extcmd *ExtCommand, con *console.Console) error {
	binPath, err := extcmd.getFileForTarget(goos, goarch)
	if err != nil {
		return err
	}
	con.RefreshActiveSession()

	if !slices.Contains(con.GetInteractive().Modules, extcmd.DependsOn) {
		return ErrExtensionDependModuleNotFound
	}

	for _, ext := range con.GetInteractive().Extensions.Extensions {
		if ext.Name == extcmd.CommandName {
			return nil
		}
	}
	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		return err
	}
	logs.Log.Infof("%s extension: %s not load, loading...", extcmd.CommandName, binPath)
	if errRegister := registerExtension(extcmd, binData, con); errRegister != nil {
		return errRegister
	}
	return nil
}

func registerExtension(ext *ExtCommand, binData []byte, con *console.Console) error {
	task, err := con.Rpc.LoadExtension(con.ActiveTarget.Context(), &implantpb.LoadExtension{
		Name:   ext.CommandName,
		Bin:    binData,
		Depend: ext.DependsOn,
		Type:   "",
	})
	if err != nil {
		return err
	}

	con.AddCallback(task.TaskId, func(msg proto.Message) {
		con.SessionLog(con.GetInteractive().SessionId).Infof("Loaded extension %s", ext.CommandName)
	})
	return nil
}

//func loadDep(goos string, goarch string, depName string, ctx *grumble.Context, con *console.Console) error {
//	depExt, ok := loadedExtensions[depName]
//	if ok {
//		depBinPath, err := depExt.getFileForTarget(goos, goarch)
//		if err != nil {
//			return err
//		}
//		depBinData, err := ioutil.ReadFile(depBinPath)
//		if err != nil {
//			return err
//		}
//		return registerExtension(goos, depExt, depBinData, ctx, con)
//	}
//	return fmt.Errorf("missing dependency %s", depName)
//}

func runExtensionCmd(ctx *grumble.Context, con *console.Console) {
	var (
		err error
		//extensionArgs []byte
		extName    string
		entryPoint string
	)
	session := con.GetInteractive()
	args := ctx.Args.StringList("arguments")
	if session == nil {
		return
	}
	var goos string
	var goarch string
	if session != nil {
		goos = session.Os.Name
		goarch = session.Os.Arch
	}

	ext, ok := loadedExtensions[ctx.Command.Name]
	if !ok {
		console.Log.Errorf("No extension command found for `%s` command\n", ctx.Command.Name)
		return
	}

	if err = loadExtension(goos, goarch, ext, con); err != nil {
		console.Log.Errorf("Could not load extension: %s\n", err)
		return
	}

	binPath, err := ext.getFileForTarget(goos, goarch)
	if err != nil {
		console.Log.Errorf("Failed to read extension file: %s\n", err)
		return
	}

	isBOF := filepath.Ext(binPath) == ".o"

	// BOFs (Beacon Object Files) are a specific kind of extensions
	// that require another extension (a COFF loader) to be present.
	// BOFs also have strongly typed arguments that need to be parsed in the proper way.
	// This block will pack both the BOF data and its arguments into a single buffer that
	// the loader will extract and load.
	if isBOF {
		// Beacon Object File -- requires a COFF loader
		//extensionArgs, err = getBOFArgs(ctx, args, binPath, ext)
		if err != nil {
			console.Log.Errorf("BOF args error: %s\n", err)
			return
		}
		//extName = ext.DependsOn
		entryPoint = loadedExtensions[extName].Entrypoint // should exist at this point
	} else {
		// Regular DLL
		//extArgs := strings.Join(args, " ")
		//extensionArgs = []byte(extArgs)
		//extName = ext.CommandName
		entryPoint = ext.Entrypoint
	}

	go func() {

	}()
	task, err := con.Rpc.ExecuteExtension(con.ActiveTarget.Context(), &implantpb.ExecuteExtension{
		Extension: ext.CommandName,
		ExecuteBinary: &implantpb.ExecuteBinary{
			Name:       ext.CommandName,
			EntryPoint: entryPoint,
			Params:     args,
			Type:       ext.DependsOn,
			Output:     true,
		},
	})
	if err != nil {
		console.Log.Errorf("Call extension error: %s\n", err.Error())
		return
	}
	con.AddCallback(task.TaskId, func(msg proto.Message) {
		resp := msg.(*implantpb.Spite).GetAssemblyResponse()
		con.SessionLog(session.SessionId).Console(string(resp.Data))
	})
}

// PrintExtOutput - Print the ext execution output
//func PrintExtOutput(extName string, commandName string, callExtension *sliverpb.CallExtension, con *console.SliverConsoleClient) {
//	if extName == commandName {
//		con.PrintInfof("Successfully executed %s\n", extName)
//	} else {
//		con.PrintInfof("Successfully executed %s (%s)\n", commandName, extName)
//	}
//	if 0 < len(string(callExtension.Output)) {
//		con.PrintInfof("Got output:\n%s\n", callExtension.Output)
//	}
//	if callExtension.Response != nil && callExtension.Response.Err != "" {
//		con.PrintErrorf("%s\n", callExtension.Response.Err)
//		return
//	}
//}

func getExtArgs(ctx *grumble.Context, args []string, _ string, ext *ExtCommand) ([]byte, error) {
	var err error
	argsBuffer := utils.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}

	// Parse BOF arguments from grumble
	missingRequiredArgs := make([]string, 0)

	// If we have an extension that expects a single string, but more than one has been parsed, combine them
	if len(ext.Arguments) == 1 && strings.Contains(ext.Arguments[0].Type, "string") && len(args) > 0 {
		// The loop below will only read the first element of args because ext.Arguments is 1
		args[0] = strings.Join(args, " ")
	}

	for _, arg := range ext.Arguments {
		// If we don't have any positional words left to consume,
		// add the remaining required extension arguments in the
		// error message.
		if len(args) == 0 {
			if !arg.Optional {
				missingRequiredArgs = append(missingRequiredArgs, "`"+arg.Name+"`")
			}
			continue
		}

		// Else pop a word from the list
		word := args[0]
		args = args[1:]

		switch arg.Type {
		case "integer":
			fallthrough
		case "int":
			val, err := strconv.Atoi(word)
			if err != nil {
				return nil, err
			}
			err = argsBuffer.AddInt(uint32(val))
			if err != nil {
				return nil, err
			}
		case "short":
			val, err := strconv.Atoi(word)
			if err != nil {
				return nil, err
			}
			err = argsBuffer.AddShort(uint16(val))
			if err != nil {
				return nil, err
			}
		case "string":
			err = argsBuffer.AddString(word)
			if err != nil {
				return nil, err
			}
		case "wstring":
			err = argsBuffer.AddWString(word)
			if err != nil {
				return nil, err
			}
		// Adding support for filepaths so we can
		// send binary data like shellcodes to BOFs
		case "file":
			data, err := os.ReadFile(word)
			if err != nil {
				return nil, err
			}
			err = argsBuffer.AddData(data)
			if err != nil {
				return nil, err
			}
		}
	}

	// Return if we have missing required arguments
	if len(missingRequiredArgs) > 0 {
		return nil, fmt.Errorf("required arguments %s were not provided", strings.Join(missingRequiredArgs, ", "))
	}

	parsedArgs, err := argsBuffer.GetBuffer()
	if err != nil {
		return nil, err
	}

	return parsedArgs, nil
}

func getBOFArgs(ctx *grumble.Context, args []string, binPath string, ext *ExtCommand) ([]byte, error) {
	var extensionArgs []byte
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return nil, err
	}

	// Now build the extension's argument buffer
	extensionArgsBuffer := utils.BOFArgsBuffer{
		Buffer: new(bytes.Buffer),
	}
	err = extensionArgsBuffer.AddString(ext.Entrypoint)
	if err != nil {
		return nil, err
	}
	err = extensionArgsBuffer.AddData(binData)
	if err != nil {
		return nil, err
	}
	parsedArgs, err := getExtArgs(ctx, args, binPath, ext)
	if err != nil {
		return nil, err
	}
	err = extensionArgsBuffer.AddData(parsedArgs)
	if err != nil {
		return nil, err
	}
	extensionArgs, err = extensionArgsBuffer.GetBuffer()
	if err != nil {
		return nil, err
	}
	return extensionArgs, nil
}

func convertOldManifest(old *ExtensionManifest_) *ExtensionManifest {
	ret := &ExtensionManifest{
		Name:            old.CommandName, //treating old command name as the manifest name to avoid weird chars mostly
		Version:         old.Version,
		ExtensionAuthor: old.ExtensionAuthor,
		OriginalAuthor:  old.OriginalAuthor,
		RepoURL:         old.RepoURL,
		RootPath:        old.RootPath,
		//only one command exists in the old manifest, so we can 'confidently' create it here
		ExtCommand: []*ExtCommand{
			{
				CommandName: old.CommandName,
				DependsOn:   old.DependsOn,
				Help:        old.Help,
				LongHelp:    old.LongHelp,
				Entrypoint:  old.Entrypoint,
				Files:       old.Files,
				Arguments:   old.Arguments,
			},
		},
	}

	for _, ext := range ret.ExtCommand {
		if dep, ok := DependOnMap[ext.DependsOn]; ok {
			ext.DependsOn = dep
		} else {
			ext.DependsOn = consts.ModuleExecutePE
		}
	}

	return ret
}

func validManifest(manifest *ExtensionManifest) error {
	if manifest.Name == "" {
		return errors.New("missing `name` field in extension manifest")
	}
	for _, extManifest := range manifest.ExtCommand {
		if extManifest.CommandName == "" {
			return errors.New("missing `command_name` field in extension manifest")
		}
		if len(extManifest.Files) == 0 {
			return errors.New("missing `files` field in extension manifest")
		}
		for _, extFiles := range extManifest.Files {
			if extFiles.OS == "" {
				return errors.New("missing `files.os` field in extension manifest")
			}
			if extFiles.Arch == "" {
				return errors.New("missing `files.arch` field in extension manifest")
			}
			extFiles.Path = utils.ResolvePath(extFiles.Path)
			if extFiles.Path == "" || extFiles.Path == "/" {
				return errors.New("missing `files.path` field in extension manifest")
			}
			extFiles.OS = strings.ToLower(extFiles.OS)
			extFiles.Arch = strings.ToLower(extFiles.Arch)
		}
		if extManifest.Help == "" {
			return errors.New("missing `help` field in extension manifest")
		}
	}
	return nil
}
