package common

import (
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func ExecuteFlagSet(f *pflag.FlagSet) {
	f.StringP("process", "n", `C:\\Windows\\System32\\notepad.exe`, "custom process path")
	f.BoolP("quit", "q", false, "disable output")
	f.Uint32P("timeout", "t", 60, "timeout, in seconds")
	f.String("arch", "", "architecture x64,x86")
}

func ParseBinaryFlags(cmd *cobra.Command) (string, []string, bool, uint32) {
	path := cmd.Flags().Arg(0)
	args := cmd.Flags().Args()[1:]
	timeout, _ := cmd.Flags().GetUint32("timeout")
	quiet, _ := cmd.Flags().GetBool("quiet")
	return path, args, !quiet, timeout
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
}

func ParseSacrificeFlags(cmd *cobra.Command) *implantpb.SacrificeProcess {
	ppid, _ := cmd.Flags().GetUint32("ppid")
	argue, _ := cmd.Flags().GetString("argue")
	isBlockDll, _ := cmd.Flags().GetBool("block_dll")
	hidden, _ := cmd.Flags().GetBool("hidden")
	disableEtw, _ := cmd.Flags().GetBool("etw")
	return NewSacrifice(ppid, hidden, isBlockDll, disableEtw, argue)
}

func CLRFlagSet(f *pflag.FlagSet) {
	f.Bool("amsi", false, "bypass AMSI")
	f.Bool("etw", false, "bypass ETW")
	f.Bool("wldp", false, "bypass WLDP")
	f.Bool("bypass-all", false, "bypass AMSI,ETW,WLDP")
}

func ParseCLRFlags(cmd *cobra.Command) map[string]string {
	bypassAmsi, _ := cmd.Flags().GetBool("amsi")
	bypassEtw, _ := cmd.Flags().GetBool("etw")
	bypassWLDP, _ := cmd.Flags().GetBool("wldp")
	bypassAll, _ := cmd.Flags().GetBool("bypass-all")

	if bypassAll {
		return map[string]string{
			"bypass_amsi": "",
			"bypass_etw":  "",
			"bypass_wldp": "",
		}
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
}

func EncryptionFlagSet(f *pflag.FlagSet) {
	f.String("encryption-type", "", "encryption type")
	f.String("encryption-key", "", "encryption key")
	f.Bool("encryption-enable", false, "whether to enable encryption")
}

func PipelineFlagSet(f *pflag.FlagSet) {
	f.StringP("listener", "l", "", "listener id")
	f.String("host", "0.0.0.0", "pipeline host, the default value is **0.0.0.0**")
	f.UintP("port", "p", 0, "pipeline port, random port is selected from the range **10000-15000**")
}

func ParsePipelineFlags(cmd *cobra.Command) (string, string, uint32) {
	listenerID, _ := cmd.Flags().GetString("listener")
	host, _ := cmd.Flags().GetString("host")
	portUint, _ := cmd.Flags().GetUint32("port")

	return listenerID, host, portUint
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
		Cert:   cert,
		Key:    key,
	}, nil
}

func ParseEncryptionFlags(cmd *cobra.Command) *clientpb.Encryption {
	encryptionType, _ := cmd.Flags().GetString("encryption-type")
	encryptionKey, _ := cmd.Flags().GetString("encryption-key")
	enable, _ := cmd.Flags().GetBool("encryption-enable")
	return &clientpb.Encryption{
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
	f.StringSliceP("modules", "m", []string{}, "Set modules e.g.: execute_exe,execute_dll")
	f.Bool("srdi", false, "enable srdi")
}

func ParseGenerateFlags(cmd *cobra.Command) (string, string, string, []string, string, int, float64, bool) {
	name, _ := cmd.Flags().GetString("profile")
	address, _ := cmd.Flags().GetString("address")
	buildTarget, _ := cmd.Flags().GetString("target")
	//buildType, _ := cmd.Flags().GetString("type")
	modules, _ := cmd.Flags().GetStringSlice("modules")
	ca, _ := cmd.Flags().GetString("ca")
	interval, _ := cmd.Flags().GetInt("interval")
	jitter, _ := cmd.Flags().GetFloat64("jitter")
	enableSRDI, _ := cmd.Flags().GetBool("srdi")
	return name, address, buildTarget, modules, ca, interval, jitter, enableSRDI
}

func ProfileSet(f *pflag.FlagSet) {
	f.String("name", "", "Overwrite profile name")
	//f.String("target", "", "Overwrite build target")
	f.String("pipeline", "", "Overwrite profile pipeline_id")
	//f.String("type", "", "Set build type")
	//f.String("proxy", "", "Overwrite proxy")
	//f.String("obfuscate", "", "Set obfuscate")
	//f.StringSlice("modules", []string{}, "Overwrite modules e.g.: execute_exe,execute_dll")
	//f.String("ca", "", "Overwrite ca")
	//f.Int("interval", 5, "Overwrite interval")
	//f.Float32("jitter", 0.2, "Overwrite jitter")
}

func ParseProfileFlags(cmd *cobra.Command) (string, string) {
	profileName, _ := cmd.Flags().GetString("name")
	//buildTarget, _ := cmd.Flags().GetString("target")
	pipelineId, _ := cmd.Flags().GetString("pipeline")
	//buildType, _ := cmd.Flags().GetString("type")
	//proxy, _ := cmd.Flags().GetString("proxy")
	//obfuscate, _ := cmd.Flags().GetString("obfuscate")
	//modules, _ := cmd.Flags().GetStringSlice("modules")
	//ca, _ := cmd.Flags().GetString("ca")
	//
	//interval, _ := cmd.Flags().GetInt("interval")
	//jitter, _ := cmd.Flags().GetFloat64("jitter")

	return profileName, pipelineId
}

func MalHttpFlagset(f *pflag.FlagSet) {
	f.Bool("ignore-cache", false, "ignore cache")
	f.String("proxy", "", "proxy")
	f.String("timeout", "", "timeout")
	f.Bool("insecure", false, "insecure")
}

func SRDIFlagSet(f *pflag.FlagSet) {
	f.String("path", "", "file path")
	//f.String("type", "", "mutant type")
	f.String("arch", "x64", "shellcode architecture, eg: x86,x64")
	f.String("platform", "win", "shellcode platform, eg: windows,linux")
	f.Uint32("id", 0, "build file id")
	f.String("function_name", "", "shellcode entrypoint")
	f.String("userdata_path", "", "user data path")
}

func ParseSRDIFlags(cmd *cobra.Command) (string, string, string, uint32, map[string]string) {
	path, _ := cmd.Flags().GetString("path")
	//typ, _ := cmd.Flags().GetString("type")
	arch, _ := cmd.Flags().GetString("arch")
	platform, _ := cmd.Flags().GetString("platform")
	id, _ := cmd.Flags().GetUint32("id")
	functionName, _ := cmd.Flags().GetString("function_name, sets the entry function name within the DLL for execution. This is critical for specifying which function will be executed when the DLL is loaded.")
	userDataPath, _ := cmd.Flags().GetString("userdata_path, allows the inclusion of user-defined data to be embedded with the shellcode during generation. This can be used to pass additional information or configuration to the payload at runtime.")
	params := map[string]string{
		"function_name": functionName,
		"userdata_path": userDataPath,
	}
	return path, arch, platform, id, params
}

func GithubFlagSet(f *pflag.FlagSet) {
	f.String("owner", "", "github owner")
	f.String("repo", "", "github repo")
	f.String("token", "", "github token")
	f.String("workflowFile", "", "github workflow file")
}

func ParseGithubFlags(cmd *cobra.Command) (string, string, string, string) {
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	token, _ := cmd.Flags().GetString("token")
	file, _ := cmd.Flags().GetString("workflowFile")
	return owner, repo, token, file
}
