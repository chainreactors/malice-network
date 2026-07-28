package exec_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	commandexec "github.com/chainreactors/malice-network/client/command/exec"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestSpawnCommandBuildsArtifactRequest(t *testing.T) {
	h := testsupport.NewHarness(t)

	err := h.Execute(
		commandexec.CommandSpawn,
		"--artifact", "beacon-windows-x64",
		"--process", `C:\Windows\System32\notepad.exe`,
		"--timeout", "45",
		"--ppid", "1234",
		"--block_dll",
		"--etw",
		"--argue", "notepad.exe -Embedding",
	)
	if err != nil {
		t.Fatalf("execute spawn failed: %v", err)
	}

	req, md := testsupport.MustSingleCall[*implantpb.ExecuteBinary](t, h, "ExecuteSpawn")
	if req.GetName() != "beacon-windows-x64" {
		t.Fatalf("artifact name = %q, want beacon-windows-x64", req.GetName())
	}
	if len(req.GetBin()) != 0 || req.GetType() != "" || req.GetArch() != 0 {
		t.Fatalf("client populated server-owned Artifact fields: %#v", req)
	}
	if req.GetProcessName() != `C:\Windows\System32\notepad.exe` {
		t.Fatalf("process = %q", req.GetProcessName())
	}
	if !req.GetOutput() || req.GetTimeout() != 45 || req.GetDelay() != 2000 {
		t.Fatalf("execution options = %#v", req)
	}
	if req.GetSacrifice().GetPpid() != 1234 ||
		!req.GetSacrifice().GetBlockDll() ||
		!req.GetSacrifice().GetEtw() ||
		req.GetSacrifice().GetArgue() != "notepad.exe -Embedding" {
		t.Fatalf("sacrifice options = %#v", req.GetSacrifice())
	}
	assertExecTaskEvent(t, h, md, commandexec.CommandSpawn)
}

func TestSpawnCommandRequiresArtifactFlag(t *testing.T) {
	h := testsupport.NewHarness(t)
	err := h.Execute(commandexec.CommandSpawn)
	if err == nil || !strings.Contains(err.Error(), `required flag(s) "artifact" not set`) {
		t.Fatalf("spawn error = %v, want missing --artifact", err)
	}
	testsupport.RequireNoPrimaryCalls(t, h)
}

func TestSpawnArtifactCompleterFiltersCompletedWindowsBeacons(t *testing.T) {
	h := testsupport.NewHarness(t)
	h.Recorder.OnArtifacts("ListArtifact", func(context.Context, any) (*clientpb.Artifacts, error) {
		return &clientpb.Artifacts{Artifacts: []*clientpb.Artifact{
			{Name: "windows-beacon", Type: consts.CommandBuildBeacon, Platform: consts.Windows, Status: consts.BuildStatusCompleted, Arch: "x64", Format: ".exe"},
			{Name: "windows-beacon-case", Type: "BEACON", Platform: "WINDOWS", Status: "COMPLETED", Arch: "x86", Format: ".dll"},
			{Name: "linux-beacon", Type: consts.CommandBuildBeacon, Platform: consts.Linux, Status: consts.BuildStatusCompleted},
			{Name: "windows-pulse", Type: consts.CommandBuildPulse, Platform: consts.Windows, Status: consts.BuildStatusCompleted},
			{Name: "windows-beacon-running", Type: consts.CommandBuildBeacon, Platform: consts.Windows, Status: consts.BuildStatusRunning},
			nil,
		}}, nil
	})

	values := spawnCompletionValues(t, commandexec.SpawnArtifactCompleter(h.Console))
	if !containsSpawnCompletion(values, "windows-beacon") || !containsSpawnCompletion(values, "windows-beacon-case") {
		t.Fatalf("completion values = %#v, want completed Windows Beacons", values)
	}
	for _, excluded := range []string{"linux-beacon", "windows-pulse", "windows-beacon-running"} {
		if containsSpawnCompletion(values, excluded) {
			t.Fatalf("completion values = %#v, should not include %q", values, excluded)
		}
	}
}

func spawnCompletionValues(t testing.TB, action carapace.Action) []string {
	t.Helper()
	data, err := json.Marshal(action.Invoke(carapace.Context{}))
	if err != nil {
		t.Fatalf("marshal completion action failed: %v", err)
	}
	var decoded struct {
		Values []struct {
			Value string `json:"value"`
		} `json:"values"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode completion action failed: %v", err)
	}
	values := make([]string, 0, len(decoded.Values))
	for _, value := range decoded.Values {
		values = append(values, value.Value)
	}
	return values
}

func containsSpawnCompletion(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
