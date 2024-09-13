package common

import (
	"github.com/spf13/pflag"
	"math"
)

func SacrificeFlagSet(f *pflag.FlagSet) {
	f.UintP("ppid", "p", 0, "pid of the process to inject into (0 means injection into ourselves)")
	f.BoolP("block_dll", "b", false, "block dll injection")
	f.StringP("argue", "a", "", "fake argue")
	f.Bool("etw", false, "disable ETW")
}

func ExecuteFlagSet(f *pflag.FlagSet) {
	f.StringP("process", "n", `C:\\Windows\\System32\\notepad.exe`, "custom process path")
	f.BoolP("quit", "q", false, "disable output")
	f.Uint32P("timeout", "t", math.MaxUint32, "timeout")
	f.String("arch", "", "architecture")
}

func TlsCertFlagSet(f *pflag.FlagSet) {
	f.String("cert_path", "", "tcp pipeline tls cert path")
	f.String("key_path", "", "tcp pipeline tls key path")
}

func PipelineFlagSet(f *pflag.FlagSet) {
	f.StringP("listener_id", "l", "", "listener id")
	f.String("host", "", "pipeline host")
	f.Uint("port", 0, "pipeline port")
}
