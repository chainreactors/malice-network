package rpc

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

func TestAddCredentialResolvesTaskWithoutSessionEnvelope(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-credential", "rpc-context-credential-pipe", true)
	task := seedRPCTestTask(t, sess, "credential")

	_, err := (&Server{}).AddCredential(context.Background(), &clientpb.Context{
		Task:  task.ToProtobuf(),
		Type:  consts.ContextCredential,
		Nonce: "nonce-cred",
		Value: output.MarshalContext(&output.CredentialContext{
			CredentialType: output.UserPassCredential,
			Target:         "server.local",
			Params: map[string]string{
				"username": "alice",
				"password": "secret",
			},
		}),
	})
	if err != nil {
		t.Fatalf("AddCredential failed: %v", err)
	}

	contexts, err := (&Server{}).GetContexts(context.Background(), &clientpb.Context{
		Task:  task.ToProtobuf(),
		Type:  consts.ContextCredential,
		Nonce: "nonce-cred",
	})
	if err != nil {
		t.Fatalf("GetContexts failed: %v", err)
	}
	if len(contexts.Contexts) != 1 {
		t.Fatalf("GetContexts count = %d, want 1", len(contexts.Contexts))
	}
	if contexts.Contexts[0].Nonce != "nonce-cred" {
		t.Fatalf("context nonce = %q, want nonce-cred", contexts.Contexts[0].Nonce)
	}
	if contexts.Contexts[0].Session == nil || contexts.Contexts[0].Session.SessionId != sess.ID {
		t.Fatalf("context session = %#v, want session %s", contexts.Contexts[0].Session, sess.ID)
	}
}

func TestAddScreenShotAcceptsRawContentWithoutMetadataValue(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-screenshot", "rpc-context-screenshot-pipe", true)
	task := seedRPCTestTask(t, sess, "screenshot")

	payload := append([]byte{0, 0, 0, 0}, []byte("jpeg-data")...)
	if _, err := (&Server{}).AddScreenShot(context.Background(), &clientpb.Context{
		Task:    task.ToProtobuf(),
		Content: payload,
		Type:    consts.ContextScreenShot,
	}); err != nil {
		t.Fatalf("AddScreenShot failed: %v", err)
	}

	contexts, err := (&Server{}).GetContexts(context.Background(), &clientpb.Context{
		Task: task.ToProtobuf(),
		Type: consts.ContextScreenShot,
	})
	if err != nil {
		t.Fatalf("GetContexts failed: %v", err)
	}
	if len(contexts.Contexts) != 1 {
		t.Fatalf("GetContexts count = %d, want 1", len(contexts.Contexts))
	}
	screenshotCtx, err := output.ToContext[*output.ScreenShotContext](contexts.Contexts[0])
	if err != nil {
		t.Fatalf("ToContext failed: %v", err)
	}
	content, err := os.ReadFile(screenshotCtx.FilePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "jpeg-data" {
		t.Fatalf("screenshot file content = %q, want jpeg-data", content)
	}
}

func TestAddUploadCreatesContextRecord(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-upload", "rpc-context-upload-pipe", true)
	task := seedRPCTestTask(t, sess, "upload")

	if _, err := (&Server{}).AddUpload(context.Background(), &clientpb.Context{
		Task: task.ToProtobuf(),
		Type: consts.ContextUpload,
		Value: output.MarshalContext(&output.UploadContext{
			FileDescriptor: &output.FileDescriptor{
				Name:       "upload.bin",
				TargetPath: "C:\\temp\\upload.bin",
				Size:       11,
			},
		}),
	}); err != nil {
		t.Fatalf("AddUpload failed: %v", err)
	}

	contexts, err := (&Server{}).GetContexts(context.Background(), &clientpb.Context{
		Task: task.ToProtobuf(),
		Type: consts.ContextUpload,
	})
	if err != nil {
		t.Fatalf("GetContexts failed: %v", err)
	}
	if len(contexts.Contexts) != 1 {
		t.Fatalf("GetContexts count = %d, want 1", len(contexts.Contexts))
	}
}

func TestSyncRequiresIdentifier(t *testing.T) {
	if _, err := (&Server{}).Sync(context.Background(), &clientpb.Sync{}); err == nil || err.Error() != "context id or task id is required" {
		t.Fatalf("Sync error = %v, want context id or task id is required", err)
	}
}

func TestSyncReturnsContextWithoutContentWhenBackingFileIsMissing(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-context-sync", "rpc-context-sync-pipe", true)
	task := seedRPCTestTask(t, sess, "download")

	model, err := db.SaveContext(&clientpb.Context{
		Task:    task.ToProtobuf(),
		Session: task.Session.ToProtobufLite(),
		Type:    consts.ContextDownload,
		Value: output.MarshalContext(&output.DownloadContext{
			FileDescriptor: &output.FileDescriptor{
				Name:       "missing.bin",
				TargetPath: "remote/missing.bin",
				FilePath:   "Z:/definitely-missing.bin",
				Size:       3,
			},
		}),
	})
	if err != nil {
		t.Fatalf("SaveContext failed: %v", err)
	}

	ctx, err := (&Server{}).Sync(context.Background(), &clientpb.Sync{ContextId: model.ID.String()})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if ctx.Id != model.ID.String() {
		t.Fatalf("context id = %q, want %q", ctx.Id, model.ID.String())
	}
	if len(ctx.Content) != 0 {
		t.Fatalf("context content length = %d, want 0", len(ctx.Content))
	}
}

func seedRPCTestTask(t testing.TB, sess *core.Session, taskType string) *core.Task {
	t.Helper()

	task := sess.NewTask(taskType, 1)
	task.Cur = 1
	task.CreatedAt = time.Now()
	task.FinishedAt = task.CreatedAt
	task.CallBy = consts.CalleeCMD

	if err := db.AddTask(task.ToProtobuf()); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := db.UpdateTaskFinish(task.TaskID()); err != nil {
		t.Fatalf("UpdateTaskFinish failed: %v", err)
	}
	return task
}
