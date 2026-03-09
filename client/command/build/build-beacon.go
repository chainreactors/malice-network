package build

import (
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/corpix/uarand"
	"strings"
	//"github.com/chainreactors/malice-network/client/assets"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func GuardrailFlagSet(f *pflag.FlagSet) {
	f.String("guardrail-ip-addresses", "", "IP address whitelist (comma-separated)")
	f.String("guardrail-usernames", "", "username whitelist (comma-separated)")
	f.String("guardrail-server-names", "", "server name whitelist (comma-separated)")
	f.String("guardrail-domains", "", "domain whitelist (comma-separated)")
	common.SetFlagSetGroup(f, "guardrail")
}

func ProxyFlagSet(f *pflag.FlagSet) {
	// Proxy flags
	f.Bool("proxy-use-env", false, "Use environment proxy settings")
	f.String("proxy-url", "", "proxy URL")
	common.SetFlagSetGroup(f, "proxy")
}

// AntiFlagSet Anti flags
func AntiFlagSet(f *pflag.FlagSet) {
	f.Bool("anti-sandbox", false, "Enable anti-sandbox detection")
	//f.Bool("anti-vm", false, "Enable anti-VM detection")
	//f.Bool("anti-debug", false, "Enable anti-debug detection")
	//f.Bool("anti-disasm", false, "Enable anti-disassembly detection")
	//f.Bool("anti-emulator", false, "Enable anti-emulator detection")
	common.SetFlagSetGroup(f, "anti")
}

// DgaFlagSet DGA flags
func DgaFlagSet(f *pflag.FlagSet) {
	f.Bool("dga-enable", false, "Enable Domain Generation Algorithm")
	f.String("dga-key", "", "DGA key")
	f.Int("dga-interval-hours", -1, "DGA generation interval in hours")
	common.SetFlagSetGroup(f, "dga")
}

func OllvmFlagSet(f *pflag.FlagSet) {
	f.Bool("ollvm", false, "Enable Ollvm")
	common.SetFlagSetGroup(f, "ollvm")
}

// BeaconFlagSet 定义所有构建相关的flag
func BeaconFlagSet(f *pflag.FlagSet) {
	// Basic profile flags
	f.String("name", "", "profile name")
	f.String("cron", "", "cron expr (e.g., '*/5 * * * * * *')")
	f.Float64("jitter", -1, "jitter value (0.0-1.0)")
	f.Int("retry", -1, "retry count")
	f.Int("max-cycles", -1, "max cycles, -1 for infinite")
	f.Bool("keepalive", false, "keepalive mode")
	f.String("encryption", "", "encryption type (aes, xor, etc.)")
	f.String("key", "", "encryption key")

	// Secure flags
	f.Bool("secure", false, "Enable secure communication")
	//f.String("secure-private-key", "", "private key for secure communication")
	//f.String("secure-public-key", "", "public key for secure communication")

	// Network target flags
	f.String("addresses", "", "Target addresses (comma-separated)")
	//f.String("rem-link", "", "REM link configuration")

	// Legacy flags for backward compatibility
	//f.String("proxy", "", "Legacy proxy override (use --proxy-url instead)")
	f.String("rem", "", "Legacy REM static link flag")
	f.Bool("auto-download", false, "Auto download artifact after build")
	f.Uint32("artifact-id", 0, "Artifact ID for pulse builds")
	//f.Uint32("relink", 0, "Relink beacon ID")

	common.SetFlagSetGroup(f, "basic")
}

func BeaconCmd(cmd *cobra.Command, con *core.Console) error {
	buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildBeacon)
	if err != nil {
		return err
	}
	return ExecuteBuild(con, buildConfig)
}

// prepareBuildConfig 准备标准构建配置
// 分层覆盖链: defaults ← profile ← archive ← individual files ← inline flags
func prepareBuildConfig(cmd *cobra.Command, con *core.Console, buildType string) (*clientpb.BuildConfig, error) {
	var err error
	profileName, _ := cmd.Flags().GetString("profile")
	target, _ := cmd.Flags().GetString("target")
	artifactId, _ := cmd.Flags().GetUint32("artifact-id")

	if target == "" {
		return nil, errors.New("require build target")
	}
	buildConfig := &clientpb.BuildConfig{
		ProfileName: profileName,
		Target:      target,
		BuildType:   buildType,
		ArtifactId:  artifactId,
	}
	buildConfig, err = parseSourceConfig(cmd, con, buildConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build config: %w", err)
	}

	// Layer 1: Load from profile (server-side)
	var implantYAML []byte
	if profileName != "" {
		profilePB, err := con.Rpc.GetProfileByName(con.Context(), &clientpb.Profile{Name: profileName})
		if err != nil {
			return nil, fmt.Errorf("failed to get profile: %w", err)
		}
		implantYAML = profilePB.ImplantConfig
		buildConfig.PreludeConfig = profilePB.PreludeConfig
		buildConfig.Resources = profilePB.Resources
	}

	// Layer 2+3: File inputs (archive < individual files)
	fileImplant, filePrelude, fileResources, err := loadBuildInputs(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to load build inputs: %w", err)
	}
	if fileImplant != nil {
		implantYAML = fileImplant
	}
	if filePrelude != nil {
		buildConfig.PreludeConfig = filePrelude
	}
	if fileResources != nil {
		buildConfig.Resources = fileResources
	}

	// Parse implant YAML into ProfileConfig
	var profile *implanttypes.ProfileConfig
	if implantYAML != nil {
		profile, err = implanttypes.LoadProfile(implantYAML)
	} else {
		profile, err = implanttypes.LoadProfile(consts.DefaultProfile)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load profile: %w", err)
	}

	// Layer 4: Inline flag overrides
	profile, err = parseBuildFlags(cmd, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build flags: %w", err)
	}

	// align implant mode with requested build type
	if profile.Implant != nil && (buildType == consts.CommandBuildBeacon || buildType == consts.CommandBuildBind) {
		profile.Implant.Mod = buildType
	}

	buildConfig.MaleficConfig, _ = profile.ToYAML()

	if err := parseLibFlag(cmd, buildConfig); err != nil {
		return nil, err
	}

	return buildConfig, nil
}

// parseBuildFlags 解析所有构建相关的flag参数
func parseBuildFlags(cmd *cobra.Command, profile *implanttypes.ProfileConfig) (*implanttypes.ProfileConfig, error) {

	//newProfile.SetDefaults()
	// Basic profile flags - only override if explicitly provided
	if cmd.Flags().Changed("cron") {
		cron, _ := cmd.Flags().GetString("cron")
		profile.Basic.Cron = cron
	}

	if cmd.Flags().Changed("jitter") {
		jitter, _ := cmd.Flags().GetFloat64("jitter")
		profile.Basic.Jitter = jitter
	}

	if cmd.Flags().Changed("retry") {
		retry, _ := cmd.Flags().GetInt("retry")
		profile.Basic.Retry = retry
	}

	if cmd.Flags().Changed("max-cycles") {
		maxCycles, _ := cmd.Flags().GetInt("max-cycles")
		profile.Basic.MaxCycles = maxCycles
	}

	if cmd.Flags().Changed("keepalive") {
		keepalive, _ := cmd.Flags().GetBool("keepalive")
		profile.Basic.Keepalive = keepalive
	}

	if cmd.Flags().Changed("encryption") {
		encryption, _ := cmd.Flags().GetString("encryption")
		profile.Basic.Encryption = encryption
	}
	if cmd.Flags().Changed("key") {
		key, _ := cmd.Flags().GetString("key")
		profile.Basic.Key = key
	}

	// secure flags - only override if explicitly provided
	if cmd.Flags().Changed("secure") {
		secureEnable, _ := cmd.Flags().GetBool("secure")
		if profile.Basic.Secure == nil {
			profile.Basic.Secure = &implanttypes.SecureProfile{}
		}
		profile.Basic.Secure.Enable = secureEnable
	}
	// proxy flags - only create if explicitly provided
	if cmd.Flags().Changed("proxy-url") || cmd.Flags().Changed("proxy-use-env") {
		if profile.Basic.Proxy == nil {
			profile.Basic.Proxy = &implanttypes.ProxyProfile{}
		}

		if cmd.Flags().Changed("proxy-url") {
			proxy, _ := cmd.Flags().GetString("proxy-url")
			profile.Basic.Proxy.URL = proxy
		}

		if cmd.Flags().Changed("proxy-use-env") {
			useEnvProxy, _ := cmd.Flags().GetBool("proxy-use-env")
			profile.Basic.Proxy.UseEnvProxy = useEnvProxy
		}
	}
	// guardrail flags
	// guardrailEnable, _ := cmd.Flags().GetBool("guardrail-enable")
	// guardrailRequireAll, _ := cmd.Flags().GetBool("guardrail-require-all")
	guardrailIPAddresses, _ := cmd.Flags().GetString("guardrail-ip-addresses")
	guardrailUsernames, _ := cmd.Flags().GetString("guardrail-usernames")
	guardrailServerNames, _ := cmd.Flags().GetString("guardrail-server-names")
	guardrailDomains, _ := cmd.Flags().GetString("guardrail-domains")
	if guardrailIPAddresses != "" {
		profile.Basic.Guardrail.IPAddresses = strings.Split(guardrailIPAddresses, ",")
	}
	if guardrailUsernames != "" {
		profile.Basic.Guardrail.Usernames = strings.Split(guardrailUsernames, ",")
	}
	if guardrailServerNames != "" {
		profile.Basic.Guardrail.ServerNames = strings.Split(guardrailServerNames, ",")
	}
	if guardrailDomains != "" {
		profile.Basic.Guardrail.Domains = strings.Split(guardrailDomains, ",")
	}
	if guardrailIPAddresses != "" ||
		guardrailUsernames != "" ||
		guardrailServerNames != "" ||
		guardrailDomains != "" {
		profile.Basic.Guardrail.Enable = true
		profile.Basic.Guardrail.RequireAll = true
	}

	// targets
	addrs, _ := cmd.Flags().GetString("addresses")
	addresses := strings.Split(addrs, ",")

	remLink, _ := cmd.Flags().GetString("rem")
	if cmd.Flags().Changed("rem") && strings.HasPrefix(addresses[0], "tcp://") {
		remAddresses := strings.Split(remLink, ",")
		for _, remAddress := range remAddresses {
			target := implanttypes.Target{}
			addr := strings.TrimPrefix(addresses[0], "tcp://")
			if !strings.Contains(addr, ":") {
				addr = addr + ":5001"
			}
			target.Address = addr
			target.REM = &implanttypes.REMProfile{
				Link: remAddress,
			}
			profile.Basic.Targets = append(profile.Basic.Targets, target)
		}
	} else if cmd.Flags().Changed("addresses") {
		for _, address := range addresses {
			target := implanttypes.Target{}
			//
			if strings.HasPrefix(address, "http://") {
				address = strings.TrimPrefix(address, "http://")
				// default port 80
				if !strings.Contains(address, ":") {
					address = address + ":80"
				}
				target.Address = address
				target.Http = &implanttypes.HttpProfile{
					Method:  "POST",
					Path:    "/",
					Version: "1.1",
					Headers: map[string]string{
						"User-Agent":   uarand.GetRandom(),
						"Content-Type": "application/octet-stream",
					},
				}
			} else if strings.HasPrefix(address, "https://") {
				address = strings.TrimPrefix(address, "https://")
				if !strings.Contains(address, ":") {
					address = address + ":443"
				}
				target.Address = address
				target.Http = &implanttypes.HttpProfile{
					Method:  "POST",
					Path:    "/",
					Version: "1.1",
					Headers: map[string]string{
						"User-Agent":   uarand.GetRandom(),
						"Content-Type": "application/octet-stream",
					},
				}
				target.TLS = &implanttypes.TLSProfile{
					Enable:           true,
					SNI:              strings.Split(address, ":")[0],
					SkipVerification: true,
				}
			} else if strings.HasPrefix(address, "tcp://") { // 走tcp的配置
				address = strings.TrimPrefix(address, "tcp://")
				if !strings.Contains(address, ":") {
					address = address + ":5001"
				}
				target.Address = address
				target.TCP = &implanttypes.TCPProfile{}
			} else if strings.HasPrefix(address, "tcp+tls://") { // 走tcp的配置
				address = strings.TrimPrefix(address, "tcp+tls://")
				if !strings.Contains(address, ":") {
					address = address + ":5001"
				}
				target.Address = address
				target.TCP = &implanttypes.TCPProfile{}
				target.TLS = &implanttypes.TLSProfile{
					Enable:           true,
					SNI:              strings.Split(address, ":")[0],
					SkipVerification: true,
				}
			} else if strings.HasPrefix(address, "mtls://") {
				// todo
			} else {
				return nil, errors.New("invalid target address: " + address)
			}
			profile.Basic.Targets = append(profile.Basic.Targets, target)
		}
	}

	// modules - only override if explicitly provided
	if cmd.Flags().Changed("modules") {
		modules, _ := cmd.Flags().GetString("modules")
		if modules != "" {
			profile.Implant.Modules = strings.Split(modules, ",")
		}
	}

	if cmd.Flags().Changed("3rd") {
		thirdModules, _ := cmd.Flags().GetString("3rd")
		if thirdModules != "" {
			profile.Implant.ThirdModules = strings.Split(thirdModules, ",")
			profile.Implant.Enable3rd = true
		}
	}

	if cmd.Flags().Changed("rem") {
		profile.Implant.Enable3rd = true
		profile.Implant.ThirdModules = append(profile.Implant.ThirdModules, "rem")
	}

	ollvm, _ := cmd.Flags().GetBool("ollvm")
	if ollvm {
		profile.Build.OLLVM = &implanttypes.OLLVMProfile{
			Enable:   true,
			BCFObf:   true,
			SplitObf: true,
			SubObf:   true,
			FCO:      true,
			ConstEnc: true,
		}
	}

	// anti configuration
	antiSandbox, _ := cmd.Flags().GetBool("anti-sandbox")
	if cmd.Flags().Changed("anti-sandbox") {
		profile.Implant.Anti = &implanttypes.AntiProfile{
			Sandbox: antiSandbox,
		}
	}

	return profile, nil
}
