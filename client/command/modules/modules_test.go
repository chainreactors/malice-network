package modules_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/chainreactors/malice-network/helper/implanttypes"
)

func TestLoadModuleFromPath(t *testing.T) {
	h := testsupport.NewHarness(t)
	path := filepath.Join(t.TempDir(), "module.dll")
	if err := os.WriteFile(path, []byte("module-binary"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := h.Execute(consts.ModuleLoadModule, "--path", path); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	req, md := testsupport.MustSingleCall[*implantpb.LoadModule](t, h, "LoadModule")
	if req.Bundle != "module.dll" {
		t.Fatalf("bundle = %q, want module.dll", req.Bundle)
	}
	if string(req.Bin) != "module-binary" {
		t.Fatalf("module binary = %q, want module-binary", req.Bin)
	}
	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)
	assertSingleTaskEvent(t, h, consts.ModuleLoadModule)
}

func TestLoadModuleFromArtifactDownloadsThenLoads(t *testing.T) {
	h := testsupport.NewHarness(t)
	h.Recorder.OnArtifact("DownloadArtifact", func(ctx context.Context, request any) (*clientpb.Artifact, error) {
		return &clientpb.Artifact{
			Name: "artifact-module.dll",
			Bin:  []byte("artifact-binary"),
		}, nil
	})

	if err := h.Execute(consts.ModuleLoadModule, "--artifact", "artifact-module.dll"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(calls))
	}
	if calls[0].Method != "DownloadArtifact" {
		t.Fatalf("first method = %s, want DownloadArtifact", calls[0].Method)
	}
	if calls[1].Method != "LoadModule" {
		t.Fatalf("second method = %s, want LoadModule", calls[1].Method)
	}

	loadReq, ok := calls[1].Request.(*implantpb.LoadModule)
	if !ok {
		t.Fatalf("load request type = %T, want *implantpb.LoadModule", calls[1].Request)
	}
	if loadReq.Bundle != "artifact-module.dll" {
		t.Fatalf("bundle = %q, want artifact-module.dll", loadReq.Bundle)
	}
	if string(loadReq.Bin) != "artifact-binary" {
		t.Fatalf("artifact binary = %q, want artifact-binary", loadReq.Bin)
	}
	assertSingleTaskEvent(t, h, consts.ModuleLoadModule)
}

func TestLoadModuleBuildUsesSelectedModules(t *testing.T) {
	h := testsupport.NewHarness(t)
	h.Recorder.OnBuildConfig("CheckSource", func(ctx context.Context, request any) (*clientpb.BuildConfig, error) {
		cfg := request.(*clientpb.BuildConfig)
		cfg.Source = consts.ArtifactFromDocker
		return cfg, nil
	})
	h.Recorder.OnArtifact("SyncBuild", func(ctx context.Context, request any) (*clientpb.Artifact, error) {
		return &clientpb.Artifact{
			Name: "built-module.dll",
			Bin:  []byte("built-module-binary"),
		}, nil
	})

	if err := h.Execute(consts.ModuleLoadModule, "--modules", "nano, execute_dll"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 3 {
		t.Fatalf("call count = %d, want 3", len(calls))
	}
	if calls[0].Method != "CheckSource" || calls[1].Method != "SyncBuild" || calls[2].Method != "LoadModule" {
		t.Fatalf("call methods = [%s %s %s], want [CheckSource SyncBuild LoadModule]", calls[0].Method, calls[1].Method, calls[2].Method)
	}

	buildReq, ok := calls[1].Request.(*clientpb.BuildConfig)
	if !ok {
		t.Fatalf("sync build request type = %T, want *clientpb.BuildConfig", calls[1].Request)
	}
	if buildReq.BuildType != consts.CommandBuildModules {
		t.Fatalf("build type = %q, want %q", buildReq.BuildType, consts.CommandBuildModules)
	}
	if buildReq.OutputType != "lib" {
		t.Fatalf("output type = %q, want lib", buildReq.OutputType)
	}
	if buildReq.Source != consts.ArtifactFromDocker {
		t.Fatalf("build source = %q, want %q", buildReq.Source, consts.ArtifactFromDocker)
	}
	profile, err := implanttypes.LoadProfileFromContent(buildReq.MaleficConfig)
	if err != nil {
		t.Fatalf("LoadProfileFromContent failed: %v", err)
	}
	if len(profile.Implant.Modules) != 2 || profile.Implant.Modules[0] != "nano" || profile.Implant.Modules[1] != "execute_dll" {
		t.Fatalf("modules = %v, want [nano execute_dll]", profile.Implant.Modules)
	}
	if profile.Implant.Enable3rd {
		t.Fatal("enable_3rd should be false for regular modules")
	}

	loadReq, ok := calls[2].Request.(*implantpb.LoadModule)
	if !ok {
		t.Fatalf("load request type = %T, want *implantpb.LoadModule", calls[2].Request)
	}
	if loadReq.Bundle != "built-module.dll" {
		t.Fatalf("bundle = %q, want built-module.dll", loadReq.Bundle)
	}
	if string(loadReq.Bin) != "built-module-binary" {
		t.Fatalf("load binary = %q, want built-module-binary", loadReq.Bin)
	}
	assertSingleTaskEvent(t, h, consts.ModuleLoadModule)
}

func TestLoadModuleBuildUsesThirdPartySelection(t *testing.T) {
	h := testsupport.NewHarness(t)

	if err := h.Execute(consts.ModuleLoadModule, "--3rd", "rem"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 3 {
		t.Fatalf("call count = %d, want 3", len(calls))
	}
	buildReq, ok := calls[1].Request.(*clientpb.BuildConfig)
	if !ok {
		t.Fatalf("sync build request type = %T, want *clientpb.BuildConfig", calls[1].Request)
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
	assertSingleTaskEvent(t, h, consts.ModuleLoadModule)
}

func TestLoadModuleBuildErrorsPropagate(t *testing.T) {
	h := testsupport.NewHarness(t)
	h.Recorder.OnArtifact("SyncBuild", func(ctx context.Context, request any) (*clientpb.Artifact, error) {
		return nil, context.DeadlineExceeded
	})

	err := h.Execute(consts.ModuleLoadModule, "--modules", "nano")
	if err == nil || err != context.DeadlineExceeded {
		t.Fatalf("Execute error = %v, want %v", err, context.DeadlineExceeded)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(calls))
	}
	if calls[0].Method != "CheckSource" || calls[1].Method != "SyncBuild" {
		t.Fatalf("call methods = [%s %s], want [CheckSource SyncBuild]", calls[0].Method, calls[1].Method)
	}
	testsupport.RequireNoSessionEvents(t, h)
}

func TestLoadModuleRejectsMutuallyExclusiveSelectors(t *testing.T) {
	h := testsupport.NewHarness(t)

	err := h.Execute(consts.ModuleLoadModule, "--modules", "nano", "--3rd", "rem")
	if err == nil {
		t.Fatal("expected mutually exclusive selector error")
	}
	if got := err.Error(); got == "" || got == context.DeadlineExceeded.Error() {
		t.Fatalf("unexpected error: %v", err)
	}

	testsupport.RequireNoPrimaryCalls(t, h)
	testsupport.RequireNoSessionEvents(t, h)
}

func TestLoadModuleRejectsMultipleInputSources(t *testing.T) {
	h := testsupport.NewHarness(t)

	err := h.Execute(consts.ModuleLoadModule, "--artifact", "module.dll", "--modules", "nano")
	if err == nil {
		t.Fatal("expected multiple input source error")
	}
	if got := err.Error(); got == "" || got == context.DeadlineExceeded.Error() {
		t.Fatalf("unexpected error: %v", err)
	}

	testsupport.RequireNoPrimaryCalls(t, h)
	testsupport.RequireNoSessionEvents(t, h)
}

func TestLoadModuleRequiresOneSource(t *testing.T) {
	h := testsupport.NewHarness(t)

	err := h.Execute(consts.ModuleLoadModule)
	if err == nil {
		t.Fatal("expected missing source error")
	}
	if got := err.Error(); got == "" || got == context.DeadlineExceeded.Error() {
		t.Fatalf("unexpected error: %v", err)
	}

	testsupport.RequireNoPrimaryCalls(t, h)
	testsupport.RequireNoSessionEvents(t, h)
}

func assertSingleTaskEvent(t testing.TB, h *testsupport.Harness, wantType string) {
	t.Helper()

	event, md := testsupport.MustSingleSessionEvent(t, h)
	if event.Task == nil {
		t.Fatal("session event task is nil")
	}
	if event.Task.Type != wantType {
		t.Fatalf("event task type = %q, want %q", event.Task.Type, wantType)
	}
	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)
}
