package build_test

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	buildcmd "github.com/chainreactors/malice-network/client/command/build"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/spf13/cobra"
)

func TestModulesCmdRequiresTargetWithoutPanic(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	cmd := &cobra.Command{Use: consts.CommandBuildModules}
	common.GenerateFlagSet(cmd.Flags())
	buildcmd.ModuleFlagSet(cmd.Flags())
	if err := cmd.Flags().Set("modules", "nano"); err != nil {
		t.Fatalf("set modules flag failed: %v", err)
	}

	err := buildcmd.ModulesCmd(cmd, h.Console)
	if err == nil || err.Error() != "require build target" {
		t.Fatalf("ModulesCmd error = %v, want require build target", err)
	}

	testsupport.RequireNoPrimaryCalls(t, h)
	testsupport.RequireNoSessionEvents(t, h)
}

func TestBuildModuleMaleficConfigClearsDefaultModulesForThirdPartySelection(t *testing.T) {
	content, err := buildcmd.BuildModuleMaleficConfig(nil, []string{" rem ", ""})
	if err != nil {
		t.Fatalf("BuildModuleMaleficConfig failed: %v", err)
	}

	profile, err := implanttypes.LoadProfileFromContent(content)
	if err != nil {
		t.Fatalf("LoadProfileFromContent failed: %v", err)
	}
	if !profile.Implant.Enable3rd {
		t.Fatal("enable_3rd should be true for third-party module selection")
	}
	if len(profile.Implant.ThirdModules) != 1 || profile.Implant.ThirdModules[0] != "rem" {
		t.Fatalf("third modules = %v, want [rem]", profile.Implant.ThirdModules)
	}
	if len(profile.Implant.Modules) != 0 {
		t.Fatalf("default modules leaked into third-party selection: %v", profile.Implant.Modules)
	}
}

func TestBuildModulesCommandConformance(t *testing.T) {
	testsupport.RunClientCases(t, []testsupport.CommandCase{
		{
			Name:    "build modules rejects mutually exclusive selectors before rpc",
			Argv:    []string{consts.CommandBuild, consts.CommandBuildModules, "--target", "x86_64-pc-windows-gnu", "--modules", "nano", "--3rd", "rem"},
			WantErr: "mutually exclusive",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "build modules forwards selected modules in malefic config",
			Argv: []string{consts.CommandBuild, consts.CommandBuildModules, "--target", "x86_64-pc-windows-gnu", "--modules", "nano, execute_dll"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnBuildConfig("CheckSource", func(ctx context.Context, request any) (*clientpb.BuildConfig, error) {
					cfg := request.(*clientpb.BuildConfig)
					cfg.Source = consts.ArtifactFromDocker
					return cfg, nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				calls := h.Recorder.Calls()
				if len(calls) != 2 {
					t.Fatalf("call count = %d, want 2", len(calls))
				}
				checkReq, ok := calls[0].Request.(*clientpb.BuildConfig)
				if !ok {
					t.Fatalf("first request type = %T, want *clientpb.BuildConfig", calls[0].Request)
				}
				if calls[0].Method != "CheckSource" {
					t.Fatalf("first method = %s, want CheckSource", calls[0].Method)
				}
				if checkReq.Target != "x86_64-pc-windows-gnu" {
					t.Fatalf("check source target = %q, want x86_64-pc-windows-gnu", checkReq.Target)
				}

				buildReq, ok := calls[1].Request.(*clientpb.BuildConfig)
				if !ok {
					t.Fatalf("second request type = %T, want *clientpb.BuildConfig", calls[1].Request)
				}
				if calls[1].Method != "Build" {
					t.Fatalf("second method = %s, want Build", calls[1].Method)
				}
				if buildReq.BuildType != consts.CommandBuildModules {
					t.Fatalf("build type = %q, want %q", buildReq.BuildType, consts.CommandBuildModules)
				}
				if buildReq.Source != consts.ArtifactFromDocker {
					t.Fatalf("build source = %q, want %q", buildReq.Source, consts.ArtifactFromDocker)
				}
				if buildReq.OutputType != "lib" {
					t.Fatalf("build output type = %q, want lib", buildReq.OutputType)
				}

				profile, err := implanttypes.LoadProfileFromContent(buildReq.MaleficConfig)
				if err != nil {
					t.Fatalf("LoadProfileFromContent failed: %v", err)
				}
				if len(profile.Implant.Modules) != 2 || profile.Implant.Modules[0] != "nano" || profile.Implant.Modules[1] != "execute_dll" {
					t.Fatalf("modules = %v, want [nano execute_dll]", profile.Implant.Modules)
				}
				if profile.Implant.Enable3rd {
					t.Fatal("enable_3rd should be false for regular module selection")
				}
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "build modules forwards third party module selection",
			Argv: []string{consts.CommandBuild, consts.CommandBuildModules, "--target", "x86_64-pc-windows-gnu", "--3rd", "rem"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				calls := h.Recorder.Calls()
				if len(calls) != 2 {
					t.Fatalf("call count = %d, want 2", len(calls))
				}
				buildReq, ok := calls[1].Request.(*clientpb.BuildConfig)
				if !ok {
					t.Fatalf("second request type = %T, want *clientpb.BuildConfig", calls[1].Request)
				}

				profile, err := implanttypes.LoadProfileFromContent(buildReq.MaleficConfig)
				if err != nil {
					t.Fatalf("LoadProfileFromContent failed: %v", err)
				}
				if !profile.Implant.Enable3rd {
					t.Fatal("enable_3rd should be true for third-party module selection")
				}
				if len(profile.Implant.ThirdModules) != 1 || profile.Implant.ThirdModules[0] != "rem" {
					t.Fatalf("third modules = %v, want [rem]", profile.Implant.ThirdModules)
				}
				if len(profile.Implant.Modules) != 0 {
					t.Fatalf("regular modules should be empty when --3rd is used, got %v", profile.Implant.Modules)
				}
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
	})
}
