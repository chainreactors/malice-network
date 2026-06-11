package generate

import (
	"reflect"
	"testing"
)

func TestConfigToArgsUsesSGNv2Flags(t *testing.T) {
	config := SGNConfig{
		Architecture:   "x64",
		Iterations:     0,
		MaxObfuscation: 0,
		Input:          "input.bin",
		Output:         "output.bin",
	}

	got := configToArgs(config)
	want := []string{
		"--input", "input.bin",
		"--out", "output.bin",
		"--arch", "64",
		"--enc", "1",
		"--max", "20",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("configToArgs() = %#v, want %#v", got, want)
	}
}

func TestConfigToArgsIncludesOptionalSGNv2Flags(t *testing.T) {
	config := SGNConfig{
		Architecture:   "x86",
		Iterations:     3,
		MaxObfuscation: 50,
		Safe:           true,
		PlainDecoder:   true,
		Asci:           true,
		BadChars:       []byte{0x00, 0x0a, 0xff},
		Verbose:        true,
		Input:          "payload.bin",
		Output:         "encoded.bin",
	}

	got := configToArgs(config)
	want := []string{
		"--input", "payload.bin",
		"--out", "encoded.bin",
		"--arch", "32",
		"--enc", "3",
		"--max", "50",
		"--safe",
		"--plain",
		"--ascii",
		"--badchars", "\\x00\\x0a\\xff",
		"--verbose",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("configToArgs() = %#v, want %#v", got, want)
	}
}

func TestConfigToArgsDoesNotEmitLegacySGNFlags(t *testing.T) {
	args := configToArgs(SGNConfig{
		Architecture: "386",
		BadChars:     []byte{0x00},
		Input:        "in.bin",
		Output:       "out.bin",
	})

	legacyFlags := map[string]struct{}{
		"-max":           {},
		"-safe":          {},
		"-plain-decoder": {},
		"-asci":          {},
		"-b":             {},
	}
	for _, arg := range args {
		if _, ok := legacyFlags[arg]; ok {
			t.Fatalf("configToArgs() emitted legacy flag %q in %#v", arg, args)
		}
	}

	if args[len(args)-1] == "in.bin" {
		t.Fatalf("configToArgs() used positional input instead of --input: %#v", args)
	}
}

func TestNormalizeSGNArchitecture(t *testing.T) {
	tests := map[string]string{
		"386":    "32",
		"32":     "32",
		"x86":    "32",
		" i386 ": "32",
		"64":     "64",
		"x64":    "64",
		"amd64":  "64",
		"":       "64",
	}

	for arch, want := range tests {
		if got := normalizeSGNArchitecture(arch); got != want {
			t.Fatalf("normalizeSGNArchitecture(%q) = %q, want %q", arch, got, want)
		}
	}
}
