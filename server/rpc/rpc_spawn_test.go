package rpc

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestExecuteSpawnDispatchesArtifactToMatchingExecutor(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		bin        []byte
		wantModule string
	}{
		{name: "exe", format: ".exe", bin: spawnTestPE(false), wantModule: consts.ModuleExecuteExe},
		{name: "dll", format: ".dll", bin: spawnTestPE(true), wantModule: consts.ModuleExecuteDll},
		{name: "shellcode", format: consts.ShellcodeFile, bin: []byte{0x90, 0xc3}, wantModule: consts.ModuleExecuteShellcode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newRPCTestEnv(t)
			sess := env.seedSession(t, "spawn-"+tt.name+"-session", "spawn-"+tt.name+"-pipe", true)
			sess.ApplyModules(&implantpb.Modules{Modules: []string{tt.wantModule}}, false)

			artifactName := "spawn-" + tt.name + "-artifact"
			spawnSaveArtifact(t, &clientpb.Artifact{
				Name:     artifactName,
				Type:     consts.CommandBuildBeacon,
				Platform: consts.Windows,
				Arch:     "x64",
				Target:   consts.TargetX64Windows,
				Format:   tt.format,
			}, tt.bin)

			sent := make(chan *clientpb.SpiteRequest, 1)
			pipelinesCh.Store(sess.PipelineID, &testRPCServerStream{
				sendMsg: func(message interface{}) error {
					req, ok := message.(*clientpb.SpiteRequest)
					if !ok {
						t.Fatalf("unexpected message type %T", message)
					}
					sent <- req
					return nil
				},
			})
			t.Cleanup(func() { pipelinesCh.Delete(sess.PipelineID) })

			task, err := (&Server{}).ExecuteSpawn(incomingSessionContext(sess.ID), &implantpb.ExecuteBinary{
				Name:        artifactName,
				Bin:         []byte("client-controlled"),
				Type:        consts.ModuleExecuteShellcode,
				ProcessName: `C:\Windows\System32\notepad.exe`,
				Output:      true,
				Arch:        uint32(consts.Mips),
				Timeout:     60,
				Delay:       123,
				Sacrifice:   &implantpb.SacrificeProcess{Ppid: 4321, BlockDll: true},
			})
			if err != nil {
				t.Fatalf("ExecuteSpawn failed: %v", err)
			}
			if task.GetType() != tt.wantModule {
				t.Fatalf("task type = %q, want %q", task.GetType(), tt.wantModule)
			}

			select {
			case spiteReq := <-sent:
				binaryReq := spiteReq.GetSpite().GetExecuteBinary()
				if binaryReq.GetName() != artifactName || binaryReq.GetType() != tt.wantModule {
					t.Fatalf("spawn request identity = %#v", binaryReq)
				}
				if string(binaryReq.GetBin()) != string(tt.bin) {
					t.Fatalf("spawn binary did not come from stored Artifact")
				}
				if binaryReq.GetArch() != uint32(consts.X86_64) {
					t.Fatalf("spawn arch = %d, want x64", binaryReq.GetArch())
				}
				if binaryReq.GetTimeout() != 60000 || binaryReq.GetDelay() != 123 || !binaryReq.GetOutput() {
					t.Fatalf("spawn execution options = %#v", binaryReq)
				}
				if binaryReq.GetSacrifice().GetPpid() != 4321 || !binaryReq.GetSacrifice().GetBlockDll() {
					t.Fatalf("spawn sacrifice options = %#v", binaryReq.GetSacrifice())
				}
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for spawn request")
			}

			deliverTaskResponse(t, sess, task.GetTaskId(), &implantpb.Spite{
				Body: &implantpb.Spite_BinaryResponse{BinaryResponse: &implantpb.BinaryResponse{}},
			})
			waitForTaskDone(t, sess.Tasks.Get(task.GetTaskId()), "spawn task")
			waitForCondition(t, 2*time.Second, func() bool {
				_, pending := sess.GetResp(task.GetTaskId())
				return !pending
			}, "spawn response handler cleanup")
		})
	}
}

func TestExecuteSpawnRejectsInvalidArtifactsAndSessions(t *testing.T) {
	tests := []struct {
		name        string
		artifact    *clientpb.Artifact
		bin         []byte
		modules     []string
		prepare     func(*core.Session, uint32)
		wantMessage string
	}{
		{
			name:        "non Beacon Artifact",
			artifact:    &clientpb.Artifact{Type: consts.CommandBuildPulse, Platform: consts.Windows, Arch: "x64", Format: ".exe"},
			bin:         spawnTestPE(false),
			modules:     []string{consts.ModuleExecuteExe},
			wantMessage: "not a Beacon",
		},
		{
			name:        "non Windows Artifact",
			artifact:    &clientpb.Artifact{Type: consts.CommandBuildBeacon, Platform: consts.Linux, Arch: "x64", Format: ".exe"},
			bin:         spawnTestPE(false),
			modules:     []string{consts.ModuleExecuteExe},
			wantMessage: "not a Windows Artifact",
		},
		{
			name:        "incomplete Artifact",
			artifact:    &clientpb.Artifact{Type: consts.CommandBuildBeacon, Platform: consts.Windows, Arch: "x64", Format: ".exe"},
			bin:         spawnTestPE(false),
			modules:     []string{consts.ModuleExecuteExe},
			prepare:     func(_ *core.Session, id uint32) { db.UpdateBuilderStatus(id, consts.BuildStatusRunning) },
			wantMessage: "not completed",
		},
		{
			name:        "unsupported format",
			artifact:    &clientpb.Artifact{Type: consts.CommandBuildBeacon, Platform: consts.Windows, Arch: "x64", Format: ".txt"},
			bin:         []byte("not an executable"),
			modules:     []string{consts.ModuleExecuteShellcode},
			wantMessage: "unsupported binary format",
		},
		{
			name:        "missing executor module",
			artifact:    &clientpb.Artifact{Type: consts.CommandBuildBeacon, Platform: consts.Windows, Arch: "x64", Format: ".dll"},
			bin:         spawnTestPE(true),
			wantMessage: "required module",
		},
		{
			name:     "non Windows session",
			artifact: &clientpb.Artifact{Type: consts.CommandBuildBeacon, Platform: consts.Windows, Arch: "x64", Format: ".exe"},
			bin:      spawnTestPE(false),
			modules:  []string{consts.ModuleExecuteExe},
			prepare: func(sess *core.Session, _ uint32) {
				sess.UpdateSysInfo(&implantpb.SysInfo{Os: &implantpb.Os{Name: consts.Linux, Arch: "amd64"}})
			},
			wantMessage: "only supported on Windows sessions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newRPCTestEnv(t)
			sess := env.seedSession(t, "reject-"+strings.ReplaceAll(tt.name, " ", "-")+"-session", "reject-"+strings.ReplaceAll(tt.name, " ", "-")+"-pipe", true)
			sess.ApplyModules(&implantpb.Modules{Modules: tt.modules}, false)

			tt.artifact.Name = "reject-" + strings.ReplaceAll(tt.name, " ", "-")
			artifactID := spawnSaveArtifact(t, tt.artifact, tt.bin)
			if tt.prepare != nil {
				tt.prepare(sess, artifactID)
			}

			_, err := (&Server{}).ExecuteSpawn(incomingSessionContext(sess.ID), &implantpb.ExecuteBinary{Name: tt.artifact.Name})
			if status.Code(err) != codes.FailedPrecondition || !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("ExecuteSpawn error = %v, want FailedPrecondition containing %q", err, tt.wantMessage)
			}
		})
	}
}

func TestExecuteSpawnRequiresArtifactName(t *testing.T) {
	for _, req := range []*implantpb.ExecuteBinary{nil, {}} {
		if _, err := (&Server{}).ExecuteSpawn(incomingSessionContext("unused"), req); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("ExecuteSpawn(%#v) error = %v, want InvalidArgument", req, err)
		}
	}
}

func spawnSaveArtifact(t testing.TB, artifact *clientpb.Artifact, content []byte) uint32 {
	t.Helper()
	model, err := db.SaveUploadedArtifact(artifact)
	if err != nil {
		t.Fatalf("SaveUploadedArtifact failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(model.Path), 0o700); err != nil {
		t.Fatalf("create Artifact directory failed: %v", err)
	}
	if err := os.WriteFile(model.Path, content, 0o600); err != nil {
		t.Fatalf("write Artifact failed: %v", err)
	}
	return model.ID
}

func spawnTestPE(dll bool) []byte {
	content := make([]byte, 128)
	copy(content[:2], "MZ")
	binary.LittleEndian.PutUint32(content[60:64], 64)
	copy(content[64:68], "PE\x00\x00")
	characteristics := uint16(0x0002)
	if dll {
		characteristics |= 0x2000
	}
	binary.LittleEndian.PutUint16(content[86:88], characteristics)
	return content
}
