package common

import (
	"github.com/spf13/pflag"
	"math"
)

func SacrificeFlagSet(f *pflag.FlagSet) {
	f.UintP("ppid", "p", 0, "spoofing parent processes, (0 means injection into ourselves)")
	f.BoolP("block_dll", "b", false, "block not microsoft dll injection")
	f.StringP("argue", "a", "", "spoofing process arguments, eg: notepad.exe ")
	f.Bool("etw", false, "disable ETW")
}

func ExecuteFlagSet(f *pflag.FlagSet) {
	f.StringP("process", "n", `C:\\Windows\\System32\\notepad.exe`, "custom process path")
	f.BoolP("quit", "q", false, "disable output")
	f.Uint32P("timeout", "t", math.MaxUint32, "timeout, in seconds")
	f.String("arch", "", "architecture amd64,x86")
}

func TlsCertFlagSet(f *pflag.FlagSet) {
	f.String("cert_path", "", "tls cert path")
	f.String("key_path", "", "tls key path")
	f.BoolP("tls", "t", false, "enable tls")
}

func PipelineFlagSet(f *pflag.FlagSet) {
	f.StringP("name", "n", "", "pipeline name")
	f.String("host", "", "pipeline host")
	f.UintP("port", "p", 0, "pipeline port")
}
