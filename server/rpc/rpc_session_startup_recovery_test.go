package rpc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func persistStartupRecoverySession(t *testing.T, sessionID string, secure bool) *models.Session {
	t.Helper()

	sessionContext := &client.SessionContext{
		SessionInfo: &client.SessionInfo{
			Os:      &implantpb.Os{Name: "windows", Arch: "amd64"},
			Process: &implantpb.Process{Name: "implant.exe"},
		},
		Argue: map[string]string{},
		Any:   map[string]interface{}{},
	}
	if secure {
		sessionContext.Secure = &implantpb.Secure{Enable: true}
		sessionContext.KeyPair = &clientpb.KeyPair{
			PublicKey:  "persisted-public-key",
			PrivateKey: "persisted-private-key",
		}
	}

	model := &models.Session{
		SessionID:   sessionID,
		RawID:       42,
		Type:        consts.TCPPipeline,
		PipelineID:  "offline-pipeline",
		ListenerID:  "offline-listener",
		Target:      "127.0.0.1",
		IsAlive:     true,
		LastCheckin: time.Now().Unix(),
		Data:        sessionContext,
	}
	if err := db.CreateOrRecoverSession(model); err != nil {
		t.Fatalf("CreateOrRecoverSession failed: %v", err)
	}
	persisted, err := db.FindSession(sessionID)
	if err != nil {
		t.Fatalf("FindSession failed: %v", err)
	}
	return persisted
}

func TestRecoverSessionDoesNotRequireRuntimePipeline(t *testing.T) {
	newRPCTestEnv(t)
	model := persistStartupRecoverySession(t, "startup-recovery-secure", true)

	recovered, err := core.RecoverSession(model)
	if err != nil {
		t.Fatalf("RecoverSession failed without listener runtime: %v", err)
	}
	t.Cleanup(recovered.Cancel)

	if recovered.SecureManager == nil {
		t.Fatal("secure session should restore its SecureManager")
	}
	if recovered.KeyPair == nil ||
		recovered.KeyPair.PublicKey != "persisted-public-key" ||
		recovered.KeyPair.PrivateKey != "persisted-private-key" {
		t.Fatalf("recovered KeyPair = %#v, want persisted key pair", recovered.KeyPair)
	}
}

func TestRecoverSessionTaskseqUsesDatabaseAndLogMaximum(t *testing.T) {
	newRPCTestEnv(t)
	model := persistStartupRecoverySession(t, "startup-recovery-taskseq", false)
	if err := db.AddTask(&clientpb.Task{
		SessionId: model.SessionID,
		TaskId:    30,
		Type:      "database-task",
		Cur:       0,
		Total:     1,
	}); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	taskDir := filepath.Join(configs.ContextPath, model.SessionID, consts.TaskPath)
	if err := os.MkdirAll(taskDir, 0o700); err != nil {
		t.Fatalf("MkdirAll task directory failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "31_0"), []byte("task log"), 0o600); err != nil {
		t.Fatalf("WriteFile task log failed: %v", err)
	}

	recovered, err := core.RecoverSession(model)
	if err != nil {
		t.Fatalf("RecoverSession failed: %v", err)
	}
	t.Cleanup(recovered.Cancel)

	if got := recovered.Taskseq.Load(); got != 31 {
		t.Fatalf("Taskseq = %d, want max(database=30, log=31)", got)
	}
	if got := recovered.NewTask("next-task", 1).Id; got != 32 {
		t.Fatalf("next task ID = %d, want 32", got)
	}
}

func TestRecoverSessionFinishedTaskHasNoResponseChannel(t *testing.T) {
	newRPCTestEnv(t)
	model := persistStartupRecoverySession(t, "startup-recovery-finished", false)
	taskID := uint32(7)
	if err := db.AddTask(&clientpb.Task{
		SessionId: model.SessionID,
		TaskId:    taskID,
		Type:      "finished-task",
		Cur:       1,
		Total:     3,
	}); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := db.UpdateTaskFinish(model.SessionID + "-7"); err != nil {
		t.Fatalf("UpdateTaskFinish failed: %v", err)
	}

	recovered, err := core.RecoverSession(model)
	if err != nil {
		t.Fatalf("RecoverSession failed: %v", err)
	}
	t.Cleanup(recovered.Cancel)

	task := recovered.Tasks.Get(taskID)
	if task == nil || !task.Finished() {
		t.Fatalf("recovered task = %#v, want finished task", task)
	}
	if _, ok := recovered.GetResp(taskID); ok {
		t.Fatal("finished task should not have an active response channel")
	}
}
