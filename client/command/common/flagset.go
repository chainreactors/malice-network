package common

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strings"
)

func ExecuteFlagSet(f *pflag.FlagSet) {
	f.StringP("process", "n", `C:\\Windows\\System32\\notepad.exe`, "custom process path")
	f.BoolP("quiet", "q", false, "disable output")
	f.Uint32P("timeout", "t", 60, "timeout, in seconds")
	f.String("arch", "", "architecture x64,x86")

	SetFlagSetGroup(f, "execute")
}

func ParseBinaryDataFlags(cmd *cobra.Command) (string, string, bool, uint32) {
	path := cmd.Flags().Arg(0)
	data := cmd.Flags().Arg(1)
	timeout, _ := cmd.Flags().GetUint32("timeout")
	quiet, _ := cmd.Flags().GetBool("quiet")
	return path, data, !quiet, timeout
}

func ParseBinaryFlags(cmd *cobra.Command) (string, []string, bool, uint32) {
	path := cmd.Flags().Arg(0)
	args := cmd.Flags().Args()[1:]
	timeout, _ := cmd.Flags().GetUint32("timeout")
	quiet, _ := cmd.Flags().GetBool("quiet")
	return path, args, !quiet, timeout
}

func ParseFullBinaryDataFlags(cmd *cobra.Command) (string, string, bool, uint32, string, string) {
	path, data, output, timeout := ParseBinaryDataFlags(cmd)
	arch, _ := cmd.Flags().GetString("arch")
	process, _ := cmd.Flags().GetString("process")
	return path, data, output, timeout, arch, process
}

func ParseFullBinaryFlags(cmd *cobra.Command) (string, []string, bool, uint32, string, string) {
	path, args, output, timeout := ParseBinaryFlags(cmd)
	arch, _ := cmd.Flags().GetString("arch")
	process, _ := cmd.Flags().GetString("process")
	return path, args, output, timeout, arch, process
}

func SacrificeFlagSet(f *pflag.FlagSet) {
	f.Uint32P("ppid", "p", 0, "spoofing parent processes, (0 means injection into ourselves)")
	f.BoolP("block_dll", "b", false, "block not microsoft dll injection")
	f.StringP("argue", "a", "", "spoofing process arguments, eg: notepad.exe ")
	f.Bool("etw", false, "disable ETW")

	SetFlagSetGroup(f, "sacrifice")
}

func ParseSacrificeFlags(cmd *cobra.Command) *implantpb.SacrificeProcess {
	ppid, _ := cmd.Flags().GetUint32("ppid")
	argue, _ := cmd.Flags().GetString("argue")
	isBlockDll, _ := cmd.Flags().GetBool("block_dll")
	hidden, _ := cmd.Flags().GetBool("hidden")
	disableEtw, _ := cmd.Flags().GetBool("etw")
	return output.NewSacrifice(ppid, hidden, isBlockDll, disableEtw, argue)
}

func CLRFlagSet(f *pflag.FlagSet) {
	f.Bool("amsi", false, "bypass AMSI")
	f.Bool("etw", false, "bypass ETW")
	f.Bool("wldp", false, "bypass WLDP")
	f.Bool("bypass-all", false, "bypass AMSI,ETW,WLDP")

	SetFlagSetGroup(f, "clr")
}

func ParseCLRFlags(cmd *cobra.Command) map[string]string {
	bypassAmsi, _ := cmd.Flags().GetBool("amsi")
	bypassEtw, _ := cmd.Flags().GetBool("etw")
	bypassWLDP, _ := cmd.Flags().GetBool("wldp")
	bypassAll, _ := cmd.Flags().GetBool("bypass-all")

	if bypassAll {
		return intermediate.NewBypassAll()
	}

	params := make(map[string]string)
	if bypassAmsi {
		params["bypass_amsi"] = ""
	}
	if bypassEtw {
		params["bypass_etw"] = ""
	}
	if bypassWLDP {
		params["bypass_wldp"] = ""
	}
	return params
}

func TlsCertFlagSet(f *pflag.FlagSet) {
	f.String("cert", "", "tls cert path")
	f.String("key", "", "tls key path")
	f.BoolP("tls", "t", false, "enable tls")

	SetFlagSetGroup(f, "tls")
}

func ArtifactFlagSet(f *pflag.FlagSet) {
	f.StringSlice("target", []string{}, "build target")
	f.String("beacon-pipeline", "", "beacon pipeline id")

	SetFlagSetGroup(f, "artifact")
}

func ParseArtifactFlags(cmd *cobra.Command) ([]string, string) {
	target, _ := cmd.Flags().GetStringSlice("target")
	beaconPipeline, _ := cmd.Flags().GetString("beacon-pipeline")
	return target, beaconPipeline
}

func PipelineFlagSet(f *pflag.FlagSet) {
	f.StringP("listener", "l", "", "listener id")
	f.String("host", "0.0.0.0", "pipeline host, the default value is **0.0.0.0**")
	f.Uint32P("port", "p", 0, "pipeline port, random port is selected from the range **10000-15000** ")
	f.String("ip", "ip", "external ip")

	SetFlagSetGroup(f, "pipeline")
}

func ParsePipelineFlags(cmd *cobra.Command) (string, string, string, uint32) {
	listenerID, _ := cmd.Flags().GetString("listener")
	host, _ := cmd.Flags().GetString("host")
	portUint, _ := cmd.Flags().GetUint32("port")
	proxy, _ := cmd.Flags().GetString("proxy")
	return listenerID, proxy, host, portUint
}

func ParseTLSFlags(cmd *cobra.Command) (*clientpb.TLS, error) {
	certPath, _ := cmd.Flags().GetString("cert_path")
	keyPath, _ := cmd.Flags().GetString("key_path")
	var err error
	var cert, key string
	if certPath != "" && keyPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			return nil, err
		}
		key, err = cryptography.ProcessPEM(keyPath)
		if err != nil {
			return nil, err
		}
	}
	return &clientpb.TLS{
		Enable: true,
		Cert: &clientpb.Cert{
			Cert: cert,
			Key:  key,
		},
	}, nil
}

func EncryptionFlagSet(f *pflag.FlagSet) {
	f.String("parser", "default", "pipeline parser")
	f.String("encryption-type", "", "encryption type")
	f.String("encryption-key", "", "encryption key")
	f.Bool("encryption-enable", false, "whether to enable encryption")
	SetFlagSetGroup(f, "encryption")
}

func ParseEncryptionFlags(cmd *cobra.Command) (string, *clientpb.Encryption) {
	encryptionType, _ := cmd.Flags().GetString("encryption-type")
	encryptionKey, _ := cmd.Flags().GetString("encryption-key")
	enable, _ := cmd.Flags().GetBool("encryption-enable")
	parser, _ := cmd.Flags().GetString("parser")
	if !enable {
		if parser == "malefic" {
			encryptionKey = "maliceofinternal"
			encryptionType = consts.CryptorAES
		} else {
			encryptionKey = "maliceofinternal"
			encryptionType = consts.CryptorXOR
		}
	}
	return parser, &clientpb.Encryption{
		Enable: enable,
		Type:   encryptionType,
		Key:    encryptionKey,
	}
}

func GenerateFlagSet(f *pflag.FlagSet) {
	f.String("profile", "", "profile name")
	f.StringP("address", "a", "", "implant address")
	f.String("target", "", "build target, specify the target arch and platform, such as  **x86_64-pc-windows-msvc**.")
	f.String("ca", "", "custom ca file")
	f.Int("interval", -1, "interval /second")
	f.Float64("jitter", -1, "jitter")
	f.String("proxy", "", "Overwrite proxy")
	f.StringP("modules", "m", "full", "Set modules e.g.: execute_exe,execute_dll")
	f.Bool("srdi", true, "enable srdi")
	f.String("resource", "", "build resource")
	SetFlagSetGroup(f, "generate")
}

func ParseGenerateFlags(cmd *cobra.Command) (string, string, string, []string, string, bool, *types.ProfileParams, string) {
	name, _ := cmd.Flags().GetString("profile")
	address, _ := cmd.Flags().GetString("address")
	buildTarget, _ := cmd.Flags().GetString("target")
	//buildType, _ := cmd.Flags().GetString("type")
	proxy, _ := cmd.Flags().GetString("proxy")
	modulesFlags, _ := cmd.Flags().GetString("modules")
	modules := strings.Split(modulesFlags, ",")
	ca, _ := cmd.Flags().GetString("ca")
	interval, _ := cmd.Flags().GetInt("interval")
	jitter, _ := cmd.Flags().GetFloat64("jitter")
	enableSRDI, _ := cmd.Flags().GetBool("srdi")
	profileParams := &types.ProfileParams{
		Interval: interval,
		Jitter:   jitter,
		Proxy:    proxy,
	}
	resource, _ := cmd.Flags().GetString("resource")
	return name, address, buildTarget, modules, ca, enableSRDI, profileParams, resource
}

func ProfileSet(f *pflag.FlagSet) {
	f.StringP("name", "n", "", "Overwrite profile name")
	//f.String("target", "", "Overwrite build target")
	f.StringP("pipeline", "p", "", "Overwrite profile basic pipeline_id")
	f.String("pulse-pipeline", "", "Overwrite profile pulse pipeline_id")
	//f.String("type", "", "Set build type")
	//f.String("obfuscate", "", "Set obfuscate")
	//f.StringSlice("modules", []string{}, "Overwrite modules e.g.: execute_exe,execute_dll")
	//f.String("ca", "", "Overwrite ca")
	//f.Int("interval", 5, "Overwrite interval")
	//f.Float32("jitter", 0.2, "Overwrite jitter")
}

func ParseProfileFlags(cmd *cobra.Command) (string, string, string) {
	profileName, _ := cmd.Flags().GetString("name")
	//buildTarget, _ := cmd.Flags().GetString("target")
	basicPipelineId, _ := cmd.Flags().GetString("pipeline")
	pulsePipelineId, _ := cmd.Flags().GetString("pulse-pipeline")

	//buildType, _ := cmd.Flags().GetString("type")
	//proxy, _ := cmd.Flags().GetString("proxy")
	//obfuscate, _ := cmd.Flags().GetString("obfuscate")
	//modules, _ := cmd.Flags().GetStringSlice("modules")
	//ca, _ := cmd.Flags().GetString("ca")
	//
	//interval, _ := cmd.Flags().GetInt("interval")
	//jitter, _ := cmd.Flags().GetFloat64("jitter")

	return profileName, basicPipelineId, pulsePipelineId
}

func MalHttpFlagset(f *pflag.FlagSet) {
	f.Bool("ignore-cache", false, "ignore cache")
	f.String("proxy", "", "proxy")
	f.String("timeout", "", "timeout")
	f.Bool("insecure", false, "insecure")

	SetFlagSetGroup(f, "mal")
}

func SRDIFlagSet(f *pflag.FlagSet) {
	f.String("path", "", "file path")
	//f.String("type", "", "mutant type")
	f.String("target", "", "shellcode build target")
	f.Uint32("id", 0, "build file id")
	f.String("function_name", "", "shellcode entrypoint")
	f.String("userdata_path", "", "user data path")

	SetFlagSetGroup(f, "srdi")
}

func ParseSRDIFlags(cmd *cobra.Command) (string, string, uint32, map[string]string) {
	path, _ := cmd.Flags().GetString("path")
	//typ, _ := cmd.Flags().GetString("type")
	target, _ := cmd.Flags().GetString("target")
	id, _ := cmd.Flags().GetUint32("id")
	functionName, _ := cmd.Flags().GetString("function_name, sets the entry function name within the DLL for execution. This is critical for specifying which function will be executed when the DLL is loaded.")
	userDataPath, _ := cmd.Flags().GetString("userdata_path, allows the inclusion of user-defined data to be embedded with the shellcode during generation. This can be used to pass additional information or configuration to the payload at runtime.")
	params := map[string]string{
		"function_name": functionName,
		"userdata_path": userDataPath,
	}
	return path, target, id, params
}

func ProxyFlagSet(f *pflag.FlagSet) {
	f.StringP("port", "p", "", "Local port to listen on")
	f.StringP("username", "u", "maliceofinternal", "Username for authentication")
	f.String("password", "maliceofinternal", "Password for authentication")
	f.String("protocol", "socks5", "Inbound protocol")
	SetFlagSetGroup(f, "proxy")
}

func GithubFlagSet(f *pflag.FlagSet) {
	f.String("owner", "", "github owner")
	f.String("repo", "", "github repo")
	f.String("token", "", "github token")
	f.String("workflowFile", "", "github workflow file")
	f.Bool("remove", false, "remove workflow")

	SetFlagSetGroup(f, "github")
}

func ParseGithubFlags(cmd *cobra.Command) (string, string, string, string, bool) {
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	token, _ := cmd.Flags().GetString("token")
	file, _ := cmd.Flags().GetString("workflowFile")
	remove, _ := cmd.Flags().GetBool("remove")
	return owner, repo, token, file, remove
}
