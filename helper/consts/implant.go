package consts

const (
	ImplantMalefic      = "malefic"
	ImplantPulse        = "pulse"
	ImplantCobaltStrike = "cobaltstrike"
)

const (
	ImplantTypeBeacon   = "beacon"
	ImplantTypeBind     = "bind"
	ImplantTypeWebshell = "webshell"
	ImplantTypeReverse  = "ssh"
)

// release type
type ReleaseType int

const (
	ReleaseWinWorkstation ReleaseType = 1 + iota
	ReleaseWinDomainController
	ReleaseWinServer
	ReleaseMacOSX
	ReleaseUbuntu
	ReleaseCentos
)

type Arch uint32

const (
	I686    Arch = 0
	X86_64  Arch = 1
	Arm     Arch = 2
	Aarch64 Arch = 3
	Mips    Arch = 4
)

func (a Arch) String() string {
	switch a {
	case I686:
		return "x86"
	case X86_64:
		return "x64"
	case Arm:
		return "arm"
	case Aarch64:
		return "arm64"
	case Mips:
		return "mips"
	}
	return ""
}

// ArchAlias 将别名映射为标准的架构名称
var ArchAlias = map[string]string{
	"x86_64": "x64",
	"amd64":  "x64",
	"x86":    "x86",
	"386":    "x86",
}

// ArchMap 将字符串映射为 Arch 枚举值
var ArchMap = map[string]Arch{
	"x64":   X86_64,
	"x86":   I686,
	"arm":   Arm,
	"arm64": Aarch64,
	"mips":  Mips,
}

const (
	Windows = "win"
	Linux   = "linux"
)

const (
	ELF           = "elf"
	PE            = "pe"
	DLL           = "dll"
	Shellcode     = ".shellcode"
	PEFile        = ".exe"
	ShellcodeFile = ".bin"
	DllFile       = ".dll"
)

var (
	WindowsVer = map[string]string{
		"5.0.2195": "2000",
		"5.1.2600": "XP",
		//"5.1.2600.1105": "XP SP1",
		//"5.1.2600.1106": "XP SP1",
		//"5.1.2600.2180": "XP SP2",
		"5.2.3790": "Server 2003/Server 2003 R2",
		//"5.2.3790.1180": "Server 2003 SP1",
		"6.0.6000":   "Vista",
		"6.0.6001":   "Vista SP1/Server2008",
		"6.0.6002":   "Vista SP2/Server2008 SP2",
		"6.1.0":      "7/Server2008 R2",
		"6.1.7600":   "7/Server2008 R2",
		"6.1.7601":   "7 SP1/Server2008 R2 SP1",
		"6.2.9200":   "8/Server2012",
		"6.3.9600":   "8.1/Server2012 R2",
		"10.0.10240": "10 1507",
		"10.0.10586": "10 1511",
		"10.0.14393": "10 1607/Server2016",
		"10.0.15063": "10 1703",
		"10.0.16299": "10 1709",
		"10.0.17134": "10 1803",
		"10.0.17763": "10 1809/Server2019",
		"10.0.18362": "10 1903",
		"10.0.18363": "10 1909",
		"10.0.19041": "10 2004/Server2004",
		"10.0.19042": "10 20H2/Server20H2",
		"10.0.19043": "10 21H2",
		"10.0.20348": "Server2022",
		"10.0.22621": "11",
		"11.0.22000": "11",
	}
)

type Target struct {
	Name string
	Arch string
	OS   string
}

var TargetMap = []*Target{
	{
		Name: "x86_64-apple-darwin",
		Arch: ArchMap["x64"].String(),
		OS:   "macos",
	},
	{
		Name: "aarch64-apple-darwin",
		Arch: ArchMap["arm64"].String(),
		OS:   "macos",
	},
	{
		Name: "x86_64-unknown-linux-gnu",
		Arch: ArchMap["x64"].String(),
		OS:   "linux",
	},
	{
		Name: "i686-unknown-linux-gnu",
		Arch: ArchMap["x86"].String(),
		OS:   "linux",
	},
	{
		Name: "x86_64-pc-windows-msvc",
		Arch: ArchMap["x64"].String(),
		OS:   "windows",
	},
	{
		Name: "i686-pc-windows-msvc",
		Arch: ArchMap["x86"].String(),
		OS:   "windows",
	},
	{
		Name: "i686-pc-windows-gnu",
		Arch: ArchMap["x86"].String(),
		OS:   "windows",
	},
	{
		Name: "x86_64-pc-windows-gnu",
		Arch: ArchMap["x64"].String(),
		OS:   "windows",
	},
	{
		Name: "x86_64-unknown-linux-musl",
		Arch: ArchMap["x64"].String(),
		OS:   "linux",
	},
	{
		Name: "i686-unknown-linux-musl",
		Arch: ArchMap["x86"].String(),
		OS:   "linux",
	},
}

func GetTargetInfo(name string) (string, string, bool) {
	for _, target := range TargetMap {
		if target.Name == name {
			return target.Arch, target.OS, true
		}
	}
	return "", "", false
}

func FormatArch(arch string) string {
	if v, found := ArchAlias[arch]; found {
		return v
	} else {
		return arch
	}
}

func MapArch(arch string) uint32 {
	arch = FormatArch(arch)
	if v, found := ArchMap[arch]; found {
		return uint32(v)
	} else {
		return 0
	}
}
