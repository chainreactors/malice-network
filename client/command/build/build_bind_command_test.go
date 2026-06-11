package build_test

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/chainreactors/malice-network/helper/implanttypes"
)

func TestBuildBindCommandUsesInlineAddressesWithoutProfile(t *testing.T) {
	testsupport.RunClientCases(t, []testsupport.CommandCase{
		{
			Name: "bind accepts protocol addresses without profile",
			Argv: []string{consts.CommandBuild, consts.CommandBuildBind,
				"--target", "x86_64-pc-windows-gnu",
				"--addresses", "tcp://127.0.0.1:5008",
			},
			Assert: assertBindBuildTarget("127.0.0.1:5008"),
		},
		{
			Name: "bind accepts host port addresses without profile",
			Argv: []string{consts.CommandBuild, consts.CommandBuildBind,
				"--target", "x86_64-pc-windows-gnu",
				"--addresses", "127.0.0.1:5008",
			},
			Assert: assertBindBuildTarget("127.0.0.1:5008"),
		},
	})
}

func assertBindBuildTarget(wantAddress string) func(testing.TB, *testsupport.Harness, error) {
	return func(t testing.TB, h *testsupport.Harness, err error) {
		t.Helper()
		calls := h.Recorder.Calls()
		if len(calls) != 2 {
			t.Fatalf("call count = %d, want 2", len(calls))
		}
		if calls[0].Method != "CheckSource" {
			t.Fatalf("first method = %s, want CheckSource", calls[0].Method)
		}
		if calls[1].Method != "Build" {
			t.Fatalf("second method = %s, want Build", calls[1].Method)
		}

		buildReq, ok := calls[1].Request.(*clientpb.BuildConfig)
		if !ok {
			t.Fatalf("build request type = %T, want *clientpb.BuildConfig", calls[1].Request)
		}
		if buildReq.ProfileName != "" {
			t.Fatalf("profile name = %q, want empty", buildReq.ProfileName)
		}
		if buildReq.BuildType != consts.CommandBuildBind {
			t.Fatalf("build type = %q, want %q", buildReq.BuildType, consts.CommandBuildBind)
		}

		profile, err := implanttypes.LoadProfileFromContent(buildReq.MaleficConfig)
		if err != nil {
			t.Fatalf("LoadProfileFromContent failed: %v", err)
		}
		if profile.Implant == nil || profile.Implant.Mod != consts.CommandBuildBind {
			t.Fatalf("implant mod = %#v, want %q", profile.Implant, consts.CommandBuildBind)
		}
		if len(profile.Basic.Targets) != 1 {
			t.Fatalf("targets count = %d, want 1", len(profile.Basic.Targets))
		}
		target := profile.Basic.Targets[0]
		if target.Address != wantAddress {
			t.Fatalf("target address = %q, want %q", target.Address, wantAddress)
		}
		if target.TCP == nil {
			t.Fatalf("target TCP profile is nil: %#v", target)
		}
	}
}
