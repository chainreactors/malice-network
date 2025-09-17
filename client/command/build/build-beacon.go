package build

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"strings"
	//"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/types"

	"github.com/chainreactors/malice-network/helper/consts"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// BeaconFlagSet 定义所有构建相关的flag
func BeaconFlagSet(f *pflag.FlagSet) {
	// Basic profile flags
	f.String("name", "", "Override profile name")
	f.String("cron", "", "Override cron expression (e.g., '*/5 * * * * * *' for every 5 seconds)")
	f.Float64("jitter", -1, "Override jitter value (0.0-1.0)")
	f.Int("init-retry", -1, "Override initial retry count")
	f.Int("server-retry", -1, "Override server retry count")
	f.Int("global-retry", -1, "Override global retry count")
	f.String("encryption", "", "Override encryption type (aes, xor, etc.)")
	f.String("key", "", "Override encryption key")

	// Proxy flags
	f.Bool("proxy-use-env", false, "Use environment proxy settings")
	f.String("proxy-url", "", "Override proxy URL")

	// Secure flags
	f.Bool("secure-enable", false, "Enable secure communication")
	f.String("secure-private-key", "", "Override private key for secure communication")
	f.String("secure-public-key", "", "Override public key for secure communication")

	// DGA flags
	f.Bool("dga-enable", false, "Enable Domain Generation Algorithm")
	f.String("dga-key", "", "Override DGA key")
	f.Int("dga-interval-hours", -1, "Override DGA generation interval in hours")

	// Guardrail flags
	f.Bool("guardrail-enable", false, "Enable environment guardrail checks")
	f.Bool("guardrail-require-all", true, "Require all guardrail conditions (AND mode) or any condition (OR mode)")
	f.String("guardrail-ip-addresses", "", "Override IP address whitelist (comma-separated)")
	f.String("guardrail-usernames", "", "Override username whitelist (comma-separated)")
	f.String("guardrail-server-names", "", "Override server name whitelist (comma-separated)")
	f.String("guardrail-domains", "", "Override domain whitelist (comma-separated)")

	// Pulse flags
	f.String("pulse-encryption", "", "Override pulse encryption type")
	f.String("pulse-key", "", "Override pulse encryption key")
	f.String("pulse-target", "", "Override pulse target address")
	f.String("pulse-protocol", "", "Override pulse protocol (http, tcp, etc.)")

	// Implant flags
	f.String("runtime", "", "Override runtime (tokio, smol, async-std)")
	f.String("mod", "", "Override implant mode (beacon, bind)")
	f.String("modules", "", "Override modules (comma-separated, e.g., 'full,execute_exe')")
	f.Bool("enable-3rd", false, "Enable 3rd party modules")
	f.String("3rd-modules", "", "Override 3rd party modules")
	f.String("3rd", "", "Third party modules for modules command")
	f.String("autorun", "", "Override autorun configuration file")

	// Network target flags
	f.String("addresses", "", "Target addresses (comma-separated)")
	f.String("rem-link", "", "REM link configuration")
	f.String("user-agent", "", "HTTP User-Agent string")

	// Metadata flags
	f.String("icon", "", "Override executable icon file")
	f.String("compile-time", "", "Override compile time")
	f.String("file-version", "", "Override file version")
	f.String("product-version", "", "Override product version")
	f.String("company-name", "", "Override company name")
	f.String("product-name", "", "Override product name")
	f.String("original-filename", "", "Override original filename")
	f.String("file-description", "", "Override file description")
	f.String("internal-name", "", "Override internal name")
	f.Bool("require-admin", false, "Require administrator privileges")
	f.Bool("require-uac", false, "Require UAC elevation")
	f.Bool("remap-path", false, "Enable path remapping")

	// Pack flags
	f.StringSlice("pack-src", nil, "Source files to pack (comma-separated)")
	f.StringSlice("pack-dst", nil, "Destination paths for packed files (comma-separated)")

	// AutoRun flags
	f.String("autorun-file", "", "AutoRun configuration file path")

	// Anti flags
	f.Bool("anti-sandbox", false, "Enable anti-sandbox detection")
	f.Bool("anti-vm", false, "Enable anti-VM detection")
	f.Bool("anti-debug", false, "Enable anti-debug detection")
	f.Bool("anti-disasm", false, "Enable anti-disassembly detection")
	f.Bool("anti-emulator", false, "Enable anti-emulator detection")

	// Build flags
	//f.Bool("zig-build", false, "Use Zig build system")
	//f.Bool("remap", false, "Enable path remapping")
	//f.String("toolchain", "", "Override build toolchain")

	// Legacy flags for backward compatibility
	f.String("proxy", "", "Legacy proxy override (use --proxy-url instead)")
	f.Int("interval", -1, "Legacy interval override (use --cron instead)")
	f.Bool("rem", false, "Legacy REM static link flag")
	f.Bool("auto-download", false, "Auto download artifact after build")
	f.Uint32("artifact-id", 0, "Artifact ID for pulse builds")
	f.Uint32("relink", 0, "Relink beacon ID")

	//SetFlagSetGroup(f, "build")
}

func BeaconCmd(cmd *cobra.Command, con *repl.Console) error {
	buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildBeacon)
	if err != nil {
		return err
	}
	executeBuild(con, buildConfig)
	return nil
}

// prepareBuildConfig 准备标准构建配置
func prepareBuildConfig(cmd *cobra.Command, con *repl.Console, buildType string) (*clientpb.BuildConfig, error) {
	var err error
	profileName, _ := cmd.Flags().GetString("profile")
	target, _ := cmd.Flags().GetString("target")
	artifact_id, _ := cmd.Flags().GetUint32("artifact_id")

	if target == "" {
		return nil, errors.New("require build target")
	}
	buildConfig := &clientpb.BuildConfig{
		ProfileName: profileName,
		Target:      target,
		BuildType:   consts.CommandBuildBeacon,
		ArtifactId:  artifact_id,
	}
	buildConfig, err = parseSourceConfig(cmd, con, buildConfig)

	// 使用新的flag解析函数
	profile, err := parseBuildFlags(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build flags: %w", err)
	}

	// Handle artifact ID for pulse builds
	//if buildType == consts.CommandBuildPulse {
	//	if buildConfig.ArtifactId == 0 && profileParams.OriginBeaconID != 0 {
	//		buildConfig.ArtifactId = profileParams.OriginBeaconID
	//	}
	//	if profileParams.RelinkBeaconID != 0 {
	//		buildConfig.ArtifactId = profileParams.RelinkBeaconID
	//	}
	//}

	//buildConfig.ParamsBytes = []byte(profileParams.String())
	buildConfig.MaleficConfig, _ = profile.ToYAML()
	//println(string(buildConfig.Bin))
	return buildConfig, nil
}

// parseBuildFlags 解析所有构建相关的flag参数
func parseBuildFlags(cmd *cobra.Command) (*types.ProfileConfig, error) {

	newProfile, _ := types.LoadProfile(consts.DefaultProfile)
	//newProfile.SetDefaults()
	// Basic profile flags - only override if explicitly provided
	if cmd.Flags().Changed("cron") {
		cron, _ := cmd.Flags().GetString("cron")
		newProfile.Basic.Cron = cron
	}

	if cmd.Flags().Changed("jitter") {
		jitter, _ := cmd.Flags().GetFloat64("jitter")
		newProfile.Basic.Jitter = jitter
	}

	if cmd.Flags().Changed("init-retry") {
		initRetry, _ := cmd.Flags().GetInt("init-retry")
		newProfile.Basic.InitRetry = initRetry
	}

	if cmd.Flags().Changed("server-retry") {
		serverRetry, _ := cmd.Flags().GetInt("server-retry")
		newProfile.Basic.ServerRetry = serverRetry
	}

	if cmd.Flags().Changed("global-retry") {
		globalRetry, _ := cmd.Flags().GetInt("global-retry")
		newProfile.Basic.GlobalRetry = globalRetry
	}

	if cmd.Flags().Changed("encryption") {
		encryption, _ := cmd.Flags().GetString("encryption")
		newProfile.Basic.Encryption = encryption
	}
	if cmd.Flags().Changed("key") {
		key, _ := cmd.Flags().GetString("key")
		newProfile.Basic.Key = key
	}

	// secure
	secureEnable, _ := cmd.Flags().GetBool("secure-enable")
	securePrivateKey, _ := cmd.Flags().GetString("secure-private-key")
	securePublicKey, _ := cmd.Flags().GetString("secure-public-key")
	if securePrivateKey != "" && securePublicKey != "" {
		newProfile.Basic.Secure = &types.SecureProfile{
			Enable:            secureEnable,
			ImplantPrivateKey: securePrivateKey,
			ServerPublicKey:   securePublicKey,
		}
	}
	// proxy flags - only create if explicitly provided
	if cmd.Flags().Changed("proxy-url") || cmd.Flags().Changed("proxy-use-env") {
		proxy, _ := cmd.Flags().GetString("proxy-url")
		use_env_proxy, _ := cmd.Flags().GetBool("proxy-use-env")
		newProfile.Basic.Proxy = &types.ProxyProfile{
			UseEnvProxy: use_env_proxy,
			URL:         proxy,
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
		newProfile.Basic.Guardrail.IPAddresses = strings.Split(guardrailIPAddresses, ",")
	}
	if guardrailUsernames != "" {
		newProfile.Basic.Guardrail.Usernames = strings.Split(guardrailUsernames, ",")
	}
	if guardrailServerNames != "" {
		newProfile.Basic.Guardrail.ServerNames = strings.Split(guardrailServerNames, ",")
	}
	if guardrailDomains != "" {
		newProfile.Basic.Guardrail.Domains = strings.Split(guardrailDomains, ",")
	}
	if guardrailIPAddresses != "" ||
		guardrailUsernames != "" ||
		guardrailServerNames != "" ||
		guardrailDomains != "" {
		newProfile.Basic.Guardrail.Enable = true
		newProfile.Basic.Guardrail.RequireAll = true
	}

	// dga flags
	//dgaEnable, _ := cmd.Flags().GetBool("dga-enable")
	//dgaKey, _ := cmd.Flags().GetString("dga-key")
	//dgaIntervalHours, _ := cmd.Flags().GetInt("dga-interval-hours")

	// targets
	// targets, _ := cmd.Flags().GetStringSlice("targets")
	addrs, _ := cmd.Flags().GetString("addresses")
	addresses := strings.Split(addrs, ",")
	ua, _ := cmd.Flags().GetString("user-agent")
	for _, address := range addresses {
		target := types.Target{}
		// 以http开头那么设置对应的http相关配置method,path,version
		if strings.HasPrefix(address, "http://") {
			address = strings.TrimPrefix(address, "http://")
			// 如果拿不到port那么默认为80
			if !strings.Contains(address, ":") {
				address = address + ":80"
			}
			target.Address = address
			target.Http = &types.HttpProfile{
				Method:  "POST",
				Path:    "/",
				Version: "1.1",
				Headers: map[string]string{
					"User-Agent":   ua,
					"Content-Type": "application/octet-stream",
				},
			}
		} else if strings.HasPrefix(address, "https://") {
			address = strings.TrimPrefix(address, "https://")
			if !strings.Contains(address, ":") {
				address = address + ":443"
			}
			target.Address = address
			target.Http = &types.HttpProfile{
				Method:  "POST",
				Path:    "/",
				Version: "1.1",
				Headers: map[string]string{
					"User-Agent":   ua,
					"Content-Type": "application/octet-stream",
				},
			}
			target.TLS = &types.TLSProfile{
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
			target.TCP = &types.TCPProfile{}
		} else if strings.HasPrefix(address, "mtls://") {
			// todo
		} else {
			return nil, errors.New("invalid target address: " + address)
		}

		newProfile.Basic.Targets = append(newProfile.Basic.Targets, target)
	}
	// rem link
	rem_link, _ := cmd.Flags().GetString("rem-link")
	remAddresses := strings.Split(rem_link, ",")
	for _, rem_address := range remAddresses {
		target := types.Target{}
		splitAddr := strings.Split(rem_address, "|")
		addr, remAddr := splitAddr[0], splitAddr[1]
		if !strings.Contains(addr, ":") {
			addr = addr + ":5001"
		}
		target.Address = addr
		target.REM = &types.REMProfile{
			Link: remAddr,
		}
		newProfile.Basic.Targets = append(newProfile.Basic.Targets, target)
	}
	// modules - only override if explicitly provided
	if cmd.Flags().Changed("modules") {
		modules, _ := cmd.Flags().GetString("modules")
		if modules != "" {
			newProfile.Implant.Modules = strings.Split(modules, ",")
		}
	}

	if cmd.Flags().Changed("3rd-modules") {
		thirdModules, _ := cmd.Flags().GetString("3rd-modules")
		if thirdModules != "" {
			newProfile.Implant.ThirdModules = strings.Split(thirdModules, ",")
			newProfile.Implant.Enable3rd = true
		}
	}
	if rem_link != "" {
		newProfile.Implant.ThirdModules = append(newProfile.Implant.ThirdModules, "rem")
	}

	// ollvm - only set if any OLLVM flag is explicitly provided
	if cmd.Flags().Changed("ollvm-bcfobf") || cmd.Flags().Changed("ollvm-splitobf") ||
		cmd.Flags().Changed("ollvm-subobf") || cmd.Flags().Changed("ollvm-fco") ||
		cmd.Flags().Changed("ollvm-constenc") {
		ollvmBcfobf, _ := cmd.Flags().GetBool("ollvm-bcfobf")
		ollvmSplitobf, _ := cmd.Flags().GetBool("ollvm-splitobf")
		ollvmSubobf, _ := cmd.Flags().GetBool("ollvm-subobf")
		ollvmFco, _ := cmd.Flags().GetBool("ollvm-fco")
		ollvmConstenc, _ := cmd.Flags().GetBool("ollvm-constenc")
		newProfile.Build.OLLVM = &types.OLLVMProfile{
			Enable:   (ollvmBcfobf || ollvmSplitobf || ollvmSubobf || ollvmFco || ollvmConstenc),
			BCFObf:   ollvmBcfobf,
			SplitObf: ollvmSplitobf,
			SubObf:   ollvmSubobf,
			FCO:      ollvmFco,
			ConstEnc: ollvmConstenc,
		}
	}
	// metadata
	icon, _ := cmd.Flags().GetString("icon")
	compileTime, _ := cmd.Flags().GetString("compile-time")
	fileVersion, _ := cmd.Flags().GetString("file-version")
	productVersion, _ := cmd.Flags().GetString("product-version")
	companyName, _ := cmd.Flags().GetString("company-name")
	productName, _ := cmd.Flags().GetString("product-name")
	originalFilename, _ := cmd.Flags().GetString("original-filename")
	fileDescription, _ := cmd.Flags().GetString("file-description")
	internalName, _ := cmd.Flags().GetString("internal-name")
	requireAdmin, _ := cmd.Flags().GetBool("require-admin")
	requireUAC, _ := cmd.Flags().GetBool("require-uac")
	remapPath, _ := cmd.Flags().GetBool("remap-path")

	if icon != "" || compileTime != "" || fileVersion != "" || productVersion != "" ||
		companyName != "" || productName != "" || originalFilename != "" ||
		fileDescription != "" || internalName != "" || requireAdmin || requireUAC || remapPath {
		newProfile.Build.Metadata = &types.MetadataProfile{
			Icon:             icon,
			CompileTime:      compileTime,
			FileVersion:      fileVersion,
			ProductVersion:   productVersion,
			CompanyName:      companyName,
			ProductName:      productName,
			OriginalFilename: originalFilename,
			FileDescription:  fileDescription,
			InternalName:     internalName,
			RequireAdmin:     requireAdmin,
			RequireUAC:       requireUAC,
		}
		if remapPath {
			newProfile.Build.Metadata.RemapPath = "true"
		}
	}

	// pack configuration
	packSources, _ := cmd.Flags().GetStringSlice("pack-src")
	packDests, _ := cmd.Flags().GetStringSlice("pack-dst")
	if len(packSources) > 0 && len(packDests) > 0 {
		if len(packSources) != len(packDests) {
			return nil, errors.New("pack-src and pack-dst must have the same number of elements")
		}
		for i, src := range packSources {
			newProfile.Implant.Pack = append(newProfile.Implant.Pack, types.PackItem{
				Src: src,
				Dst: packDests[i],
			})
		}
	}

	// autorun configuration
	autorunFile, _ := cmd.Flags().GetString("autorun-file")
	if autorunFile != "" {
		newProfile.Implant.AutoRun = autorunFile
	}

	// anti configuration
	antiSandbox, _ := cmd.Flags().GetBool("anti-sandbox")
	antiVM, _ := cmd.Flags().GetBool("anti-vm")
	antiDebug, _ := cmd.Flags().GetBool("anti-debug")
	antiDisasm, _ := cmd.Flags().GetBool("anti-disasm")
	antiEmulator, _ := cmd.Flags().GetBool("anti-emulator")
	if antiSandbox || antiVM || antiDebug || antiDisasm || antiEmulator {
		newProfile.Implant.Anti = &types.AntiProfile{
			Sandbox:  antiSandbox,
			VM:       antiVM,
			Debug:    antiDebug,
			Disasm:   antiDisasm,
			Emulator: antiEmulator,
		}
	}

	//// Set autorun file
	//if newProfile.Implant.AutoRun != "" {
	//	profileParams.AutoRunFile = newProfile.Implant.AutoRun
	//}
	//
	//// Set auto download flag
	//autoDownload, _ := cmd.Flags().GetBool("auto-download")
	//profileParams.AutoDownload = autoDownload

	return newProfile, nil
}
