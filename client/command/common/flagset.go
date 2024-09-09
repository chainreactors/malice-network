package common

import (
	"github.com/spf13/pflag"
)

func SacrificeFlagSet(f *pflag.FlagSet) {
	f.UintP("ppid", "p", 0, "pid of the process to inject into (0 means injection into ourselves)")
	f.BoolP("block_dll", "b", false, "block dll injection")
	f.StringP("process", "n", "C:\\\\Windows\\\\System32\\\\notepad.exe", "custom process path")
	f.StringP("argue", "a", "", "argue")
}

func ExecuteFlagSet(f *pflag.FlagSet) {
	f.BoolP("output", "o", true, "capture command output")
}

func TlsCertFlagSet(f *pflag.FlagSet) {
	f.String("cert_path", "", "tcp pipeline tls cert path")
	f.String("key_path", "", "tcp pipeline tls key path")
}

func PipelineFlagSet(f *pflag.FlagSet) {
	f.String("name", "", "pipeline name")
	f.String("host", "", "pipeline host")
	f.Uint("port", 0, "pipeline port")
}
