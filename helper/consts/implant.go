package consts

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

// Malefic Error
const (
	MaleficErrorPanic uint32 = 1 + iota
	MaleficErrorUnpackError
	MaleficErrorMissbody
	MaleficErrorModuleError
	MaleficErrorModuleNotFound
	MaleficErrorTaskError
	MaleficErrorTaskNotFound
	MaleficErrorTaskOperatorNotFound
	MaleficErrorExtensionNotFound
	MaleficErrorUnexceptBody
)

func GetWindowsVer(ver string) string {
	return WindowsVer[ver]
}

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

	WindowsArch = map[string]string{
		"x86_64": "amd64",
		"x86":    "386",
	}
)

func GetWindowsArch(arch string) string {
	if v, found := WindowsArch[arch]; found {
		return v
	} else {
		return arch
	}
}
