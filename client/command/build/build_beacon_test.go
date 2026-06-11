package build

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/spf13/cobra"
)

func TestParseBuildFlagsAddressesOverrideExistingTargets(t *testing.T) {
	profile, err := implanttypes.LoadProfile(consts.DefaultProfile)
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}
	profile.Basic.Targets = []implanttypes.Target{
		{
			Address: "old.example:443",
			TLS: &implanttypes.TLSProfile{
				Enable: true,
			},
		},
	}

	cmd := &cobra.Command{Use: "beacon"}
	BeaconFlagSet(cmd.Flags())
	if err := cmd.ParseFlags([]string{"--addresses", "tcp://127.0.0.1:5001"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	profile, err = parseBuildFlags(cmd, profile, consts.CommandBuildBeacon)
	if err != nil {
		t.Fatalf("parseBuildFlags failed: %v", err)
	}

	if len(profile.Basic.Targets) != 1 {
		t.Fatalf("targets count = %d, want 1", len(profile.Basic.Targets))
	}
	target := profile.Basic.Targets[0]
	if target.Address != "127.0.0.1:5001" {
		t.Fatalf("target address = %q, want %q", target.Address, "127.0.0.1:5001")
	}
	if target.TCP == nil {
		t.Fatal("expected tcp target to be set")
	}
	if target.TLS != nil || target.Http != nil || target.REM != nil {
		t.Fatalf("unexpected target protocol mix: %#v", target)
	}
}

func TestParseBuildFlagsRemAddressesOverrideExistingTargets(t *testing.T) {
	profile, err := implanttypes.LoadProfile(consts.DefaultProfile)
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}
	profile.Basic.Targets = []implanttypes.Target{
		{
			Address: "old.example:5001",
			TCP:     &implanttypes.TCPProfile{},
		},
	}

	cmd := &cobra.Command{Use: "beacon"}
	BeaconFlagSet(cmd.Flags())
	if err := cmd.ParseFlags([]string{
		"--addresses", "tcp://127.0.0.1:5001",
		"--rem", "tcp://rem-a,tcp://rem-b",
	}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	profile, err = parseBuildFlags(cmd, profile, consts.CommandBuildBeacon)
	if err != nil {
		t.Fatalf("parseBuildFlags failed: %v", err)
	}

	if len(profile.Basic.Targets) != 2 {
		t.Fatalf("targets count = %d, want 2", len(profile.Basic.Targets))
	}
	if profile.Basic.Targets[0].Address != "127.0.0.1:5001" || profile.Basic.Targets[1].Address != "127.0.0.1:5001" {
		t.Fatalf("unexpected rem target addresses: %#v", profile.Basic.Targets)
	}
	if profile.Basic.Targets[0].REM == nil || profile.Basic.Targets[0].REM.Link != "tcp://rem-a" {
		t.Fatalf("first rem target = %#v, want rem-a", profile.Basic.Targets[0])
	}
	if profile.Basic.Targets[1].REM == nil || profile.Basic.Targets[1].REM.Link != "tcp://rem-b" {
		t.Fatalf("second rem target = %#v, want rem-b", profile.Basic.Targets[1])
	}
}
