package common

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/errs"
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

func ImportSet(f *pflag.FlagSet) {
	f.String("cert", "", "tls cert path")
	f.String("key", "", "tls key path")
	f.String("ca-cert", "", "tls ca cert path")
}

func SelfSignedFlagSet(f *pflag.FlagSet) {
	f.String("CN", "", "Certificate Common Name (CN)")
	f.String("O", "", "Certificate Organization (O)")
	f.String("C", "", "Certificate Country (C)")
	f.String("L", "", "Certificate Locality/City (L)")
	f.String("OU", "", "Certificate Organizational Unit (OU)")
	f.String("ST", "", "Certificate State/Province (ST)")
	f.String("validity", "365", "Certificate validity period in days")
	SetFlagSetGroup(f, "cert")
}

func TlsCertFlagSet(f *pflag.FlagSet) {
	f.String("cert", "", "tls cert path")
	f.String("key", "", "tls key path")
	f.BoolP("tls", "t", false, "enable tls")
	f.String("cert-name", "", "certificate name")
	f.Bool("auto-cert", false, "auto cert by let's encrypt")
	f.String("domain", "", "auto cert domain")
	SetFlagSetGroup(f, "tls")
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

func ParseTLSFlags(cmd *cobra.Command) (*clientpb.TLS, string, error) {
	certPath, _ := cmd.Flags().GetString("cert")
	keyPath, _ := cmd.Flags().GetString("key")
	autoCert, _ := cmd.Flags().GetBool("auto_cert")
	domain, _ := cmd.Flags().GetString("domain")
	if autoCert && domain == "" {
		return nil, "", errs.ErrNullDomain
	}
	var err error
	var cert, key string
	if certPath != "" && keyPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			return nil, "", err
		}
		key, err = cryptography.ProcessPEM(keyPath)
		if err != nil {
			return nil, "", err
		}
	}
	certificateName, _ := cmd.Flags().GetString("cert-name")
	return &clientpb.TLS{
		Enable: true,
		Cert: &clientpb.Cert{
			Cert: cert,
			Key:  key,
		},
		AutoCert: autoCert,
		Domain:   domain,
	}, certificateName, nil
}

func EncryptionFlagSet(f *pflag.FlagSet) {
	f.String("parser", "default", "pipeline parser")
	f.String("encryption-type", "", "encryption type")
	f.String("encryption-key", "", "encryption key")
	SetFlagSetGroup(f, "encryption")
}

func ParseEncryptionFlags(cmd *cobra.Command) (string, []*clientpb.Encryption) {
	encryptionType, _ := cmd.Flags().GetString("encryption-type")
	encryptionKey, _ := cmd.Flags().GetString("encryption-key")
	parser, _ := cmd.Flags().GetString("parser")
	return parser, []*clientpb.Encryption{&clientpb.Encryption{
		Type: encryptionType,
		Key:  encryptionKey,
	}}
}

func GenerateFlagSet(f *pflag.FlagSet) {
	f.String("profile", "", "profile name")
	f.StringP("address", "a", "", "implant address")
	f.String("target", "", "build target, specify the target arch and platform, such as  **x86_64-pc-windows-msvc**.")
	//f.String("ca", "", "custom ca file")
	f.Int("interval", -1, "interval /second")
	f.Float64("jitter", -1, "jitter")
	f.String("proxy", "", "Overwrite proxy")
	f.StringP("modules", "m", "full", "Set modules e.g.: execute_exe,execute_dll")
	f.String("source", "", "build source, docker|action|saas")
	SetFlagSetGroup(f, "generate")
}

func ParseGenerateFlags(cmd *cobra.Command) *clientpb.BuildConfig {
	name, _ := cmd.Flags().GetString("profile")
	address, _ := cmd.Flags().GetString("address")
	buildTarget, _ := cmd.Flags().GetString("target")
	//buildType, _ := cmd.Flags().GetString("type")
	proxy, _ := cmd.Flags().GetString("proxy")
	modulesFlags, _ := cmd.Flags().GetString("modules")
	modules := strings.Split(modulesFlags, ",")
	//ca, _ := cmd.Flags().GetString("ca")
	interval, _ := cmd.Flags().GetInt("interval")
	jitter, _ := cmd.Flags().GetFloat64("jitter")
	source, _ := cmd.Flags().GetString("source")
	buildConfig := &clientpb.BuildConfig{
		ProfileName: name,
		MaleficHost: address,
		Target:      buildTarget,
		Modules:     modules,
		Proxy:       proxy,
		Source:      source,
	}
	artifactID, err := cmd.Flags().GetUint32("artifact-id")
	if err != nil {
	}
	pulse, err := cmd.Flags().GetUint32("pulse")
	if err != nil {
	}
	profileParams := &types.ProfileParams{
		Interval: interval,
		Jitter:   jitter,
		Proxy:    proxy,
	}
	if artifactID != 0 {
		profileParams.OriginBeaconID = artifactID
		buildConfig.ArtifactId = artifactID
	}
	if pulse != 0 {
		buildConfig.ArtifactId = pulse
		profileParams.RelinkBeaconID = pulse
	}
	buildConfig.Params = profileParams.String()
	return buildConfig
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

func ParseGithubFlags(cmd *cobra.Command) *clientpb.GithubWorkflowConfig {
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	token, _ := cmd.Flags().GetString("token")
	file, _ := cmd.Flags().GetString("workflowFile")
	remove, _ := cmd.Flags().GetBool("remove")

	githubConfig := &clientpb.GithubWorkflowConfig{
		Owner:      owner,
		Repo:       repo,
		Token:      token,
		WorkflowId: file,
		IsRemove:   remove,
	}
	if githubConfig.Owner == "" || githubConfig.Repo == "" || githubConfig.Token == "" {
		setting, err := assets.GetSetting()
		if err != nil {
			logs.Log.Errorf("get github setting error %v", err)
			return setting.Github.ToProtobuf()
		}
	}

	return githubConfig
}

// ParseSelfSignFlags parses the self-signed certificate related flags from the command and returns a CertificateSubject proto message.
func ParseSelfSignFlags(cmd *cobra.Command) *clientpb.CertificateSubject {
	cn, _ := cmd.Flags().GetString("CN")
	o, _ := cmd.Flags().GetString("O")
	c, _ := cmd.Flags().GetString("C")
	l, _ := cmd.Flags().GetString("L")
	ou, _ := cmd.Flags().GetString("OU")
	st, _ := cmd.Flags().GetString("ST")
	validity, _ := cmd.Flags().GetString("validity")
	return &clientpb.CertificateSubject{
		Cn:       cn,
		O:        o,
		C:        c,
		L:        l,
		Ou:       ou,
		St:       st,
		Validity: validity,
	}
}

func ParseImportCertFlags(cmd *cobra.Command) (*clientpb.TLS, error) {
	certPath, _ := cmd.Flags().GetString("cert")
	keyPath, _ := cmd.Flags().GetString("key")
	caPath, _ := cmd.Flags().GetString("ca-cert")

	var err error
	var cert, key, ca string
	if certPath != "" && keyPath != "" && caPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			return nil, err
		}
		key, err = cryptography.ProcessPEM(keyPath)
		if err != nil {
			return nil, err
		}
		ca, err = cryptography.ProcessPEM(caPath)
		if err != nil {
			return nil, err
		}
	}
	return &clientpb.TLS{
		Cert: &clientpb.Cert{
			Cert: cert,
			Key:  key,
		},
		Ca: &clientpb.Cert{
			Cert: ca,
		},
	}, nil
}
