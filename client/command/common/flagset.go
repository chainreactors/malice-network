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
	f.UintP("ppid", "p", 0, "spoofing parent processes, (0 means injection into ourselves)")
	f.BoolP("block_dll", "b", false, "block not microsoft dll injection")
	f.StringP("argue", "a", "", "spoofing process arguments, eg: notepad.exe ")
	f.Bool("etw", false, "disable ETW")
}

func ParseSacrificeFlags(cmd *cobra.Command) (*implantpb.SacrificeProcess, error) {
	ppid, _ := cmd.Flags().GetUint("ppid")
	argue, _ := cmd.Flags().GetString("argue")
	isBlockDll, _ := cmd.Flags().GetBool("block_dll")
	hidden, _ := cmd.Flags().GetBool("hidden")
	disableEtw, _ := cmd.Flags().GetBool("etw")
	return NewSacrifice(int64(ppid), hidden, isBlockDll, disableEtw, argue), nil
}

func CLRFlagSet(f *pflag.FlagSet) {
	f.Bool("amsi", false, "disable AMSI")
	f.Bool("etw", false, "disable ETW")
}

func ParseCLRFlags(cmd *cobra.Command) (bool, bool) {
	disableAmsi, _ := cmd.Flags().GetBool("amsi")
	disableEtw, _ := cmd.Flags().GetBool("etw")
	return disableAmsi, disableEtw
}

func TlsCertFlagSet(f *pflag.FlagSet) {
	f.String("cert", "", "tls cert path")
	f.String("key", "", "tls key path")
	f.BoolP("tls", "t", false, "enable tls")
}

func PipelineFlagSet(f *pflag.FlagSet) {
	f.StringP("listener", "l", "", "listener id")
	f.String("host", "0.0.0.0", "pipeline host")
	f.UintP("port", "p", 0, "pipeline port")
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

func GenerateFlagSet(f *pflag.FlagSet) {
	f.String("profile_name", "", "profile name")
	f.StringP("ip", "i", "", "build ip")
	f.StringP("format", "f", "", "build type")
	f.String("target", "", "build target")
	f.String("ca", "", "Set ca")
	f.String("interval", "", "interval")
	f.StringSliceP("modules", "m", []string{}, "Set modules e.g.: execute_exe,execute_dll")
	f.String("jitter", "", "jitter")
}

func ParseGenerateFlags(cmd *cobra.Command) (string, string, string, string, []string, string, string, string) {
	name, _ := cmd.Flags().GetString("profile_name")
	url, _ := cmd.Flags().GetString("ip")
	target, _ := cmd.Flags().GetString("target")
	buildType, _ := cmd.Flags().GetString("format")
	modules, _ := cmd.Flags().GetStringSlice("modules")
	ca, _ := cmd.Flags().GetString("ca")
	interval, _ := cmd.Flags().GetString("interval")
	jitter, _ := cmd.Flags().GetString("jitter")
	return name, url, target, buildType, modules, ca, interval, jitter
}

func ProfileSet(f *pflag.FlagSet) {
	f.String("name", "", "Set profile name")
	f.String("target", "", "Set build target")
	f.String("pipeline_id", "", "Set profile pipeline_id")
	f.String("type", "", "Set build type")
	f.String("proxy", "", "Set proxy")
	f.String("obfuscate", "", "Set obfuscate")
	f.StringSlice("modules", []string{}, "Set modules e.g.: execute_exe,execute_dll")
	f.String("ca", "", "Set ca")

	f.Int("interval", 5, "Set interval")
	f.Float32("jitter", 0.2, "Set jitter")
}

func ParseProfileFlags(cmd *cobra.Command) (string, string, string, string, string, string, []string, string, int, float32) {
	profileName, _ := cmd.Flags().GetString("name")
	buildTarget, _ := cmd.Flags().GetString("target")
	pipelineId, _ := cmd.Flags().GetString("pipeline_id")
	buildType, _ := cmd.Flags().GetString("type")
	proxy, _ := cmd.Flags().GetString("proxy")
	obfuscate, _ := cmd.Flags().GetString("obfuscate")
	modules, _ := cmd.Flags().GetStringSlice("modules")
	ca, _ := cmd.Flags().GetString("ca")

	interval, _ := cmd.Flags().GetInt("interval")
	jitter, _ := cmd.Flags().GetFloat32("jitter")

	return profileName, buildTarget, pipelineId, buildType, proxy, obfuscate, modules, ca, interval, jitter
}

func RegistryFlagSet(f *pflag.FlagSet) {
	f.String("hive", "", "Registry hive (e.g., HKEY_LOCAL_MACHINE)")
	f.String("path", "", "Path within the registry hive")
	f.String("key", "", "Key to query or modify")
}

func ParseRegistryFlags(cmd *cobra.Command) (string, string, string) {
	hive, _ := cmd.Flags().GetString("hive")
	path, _ := cmd.Flags().GetString("path")
	key, _ := cmd.Flags().GetString("key")
	return hive, path, key
}

func MalHttpFlagset(f *pflag.FlagSet) {
	f.Bool("ignore-cache", false, "ignore cache")
	f.String("proxy", "", "proxy")
	f.String("timeout", "", "timeout")
	f.Bool("insecure", false, "insecure")
}
