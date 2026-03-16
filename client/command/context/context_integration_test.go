package context_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/assets"
	ctxcmd "github.com/chainreactors/malice-network/client/command/context"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/testsupport"
	"github.com/spf13/cobra"
)

func TestDownloadContextLifecycleIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "ctx-pipe"), true)
	serverSession := h.SeedSession(t, "ctx-session", "ctx-pipe", true)
	task := h.SeedTask(t, serverSession, "download")
	clientHarness := testsupport.NewClientHarness(t, h)

	clientSession := clientHarness.Console.Sessions[serverSession.ID]
	if clientSession == nil {
		t.Fatalf("expected client session cache to include seeded session")
	}

	filePath, err := h.WriteTempFile("download.txt", []byte("download-body"))
	if err != nil {
		t.Fatalf("WriteTempFile failed: %v", err)
	}
	added, err := ctxcmd.AddDownload(clientHarness.Console, clientSession, task.ToProtobuf(), &output.FileDescriptor{
		Name:       "download.txt",
		TargetPath: "C:\\temp\\download.txt",
		FilePath:   filePath,
		Size:       int64(len("download-body")),
	})
	if err != nil {
		t.Fatalf("AddDownload failed: %v", err)
	}
	if !added {
		t.Fatal("expected AddDownload to report success")
	}

	downloads, err := ctxcmd.GetDownloads(clientHarness.Console)
	if err != nil {
		t.Fatalf("GetDownloads failed: %v", err)
	}
	if len(downloads) != 1 {
		t.Fatalf("GetDownloads count = %d, want 1", len(downloads))
	}

	downloadCmd := &cobra.Command{Use: "download"}
	downloadOutput := testsupport.CaptureOutput(func() {
		err = ctxcmd.GetDownloadsCmd(downloadCmd, clientHarness.Console)
	})
	if err != nil {
		t.Fatalf("GetDownloadsCmd failed: %v", err)
	}
	if !strings.Contains(downloadOutput, "download.txt") || !strings.Contains(downloadOutput, serverSession.ID) {
		t.Fatalf("download output missing expected values:\n%s", downloadOutput)
	}

	contextCmd := &cobra.Command{Use: "context"}
	contextOutput := testsupport.CaptureOutput(func() {
		err = ctxcmd.ListContexts(contextCmd, clientHarness.Console)
	})
	if err != nil {
		t.Fatalf("ListContexts failed: %v", err)
	}
	if !strings.Contains(contextOutput, serverSession.ID) || !strings.Contains(contextOutput, consts.ContextDownload) {
		t.Fatalf("context output missing expected values:\n%s", contextOutput)
	}

	contexts, err := ctxcmd.GetContextsByType(clientHarness.Console, consts.ContextDownload)
	if err != nil {
		t.Fatalf("GetContextsByType failed: %v", err)
	}
	if len(contexts.Contexts) != 1 {
		t.Fatalf("GetContextsByType count = %d, want 1", len(contexts.Contexts))
	}
}

func TestGetContextsByTaskRespectsTypeFilter(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "ctx-task-pipe"), true)
	serverSession := h.SeedSession(t, "ctx-task-session", "ctx-task-pipe", true)
	task := h.SeedTask(t, serverSession, "task-filter")
	h.SeedDownloadContext(t, task, "task-filter.txt", []byte("download"))
	h.SeedCredentialContext(t, task, "host.local", map[string]string{
		"username": "alice",
		"password": "secret",
	})
	clientHarness := testsupport.NewClientHarness(t, h)

	contexts, err := ctxcmd.GetContextsByTask(clientHarness.Console, consts.ContextDownload, task.ToProtobuf())
	if err != nil {
		t.Fatalf("GetContextsByTask failed: %v", err)
	}
	if len(contexts.Contexts) != 1 {
		t.Fatalf("GetContextsByTask count = %d, want 1", len(contexts.Contexts))
	}
	if contexts.Contexts[0].Type != consts.ContextDownload {
		t.Fatalf("GetContextsByTask type = %s, want %s", contexts.Contexts[0].Type, consts.ContextDownload)
	}
}

func TestGetCredentialsCmdDoesNotCollapseDistinctTargets(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "ctx-cred-pipe"), true)
	serverSession := h.SeedSession(t, "ctx-cred-session", "ctx-cred-pipe", true)
	task := h.SeedTask(t, serverSession, "credential-filter")
	clientHarness := testsupport.NewClientHarness(t, h)

	_, err := clientHarness.Console.Rpc.AddContext(clientHarness.Console.Context(), &clientpb.Context{
		Session: task.Session.ToProtobufLite(),
		Task:    task.ToProtobuf(),
		Type:    consts.ContextCredential,
		Value: output.MarshalContext(&output.CredentialContext{
			CredentialType: output.UserPassCredential,
			Target:         "server-a.local",
			Params: map[string]string{
				"username": "alice",
				"password": "secret",
			},
		}),
	})
	if err != nil {
		t.Fatalf("AddContext first credential failed: %v", err)
	}
	_, err = clientHarness.Console.Rpc.AddContext(clientHarness.Console.Context(), &clientpb.Context{
		Session: task.Session.ToProtobufLite(),
		Task:    task.ToProtobuf(),
		Type:    consts.ContextCredential,
		Value: output.MarshalContext(&output.CredentialContext{
			CredentialType: output.UserPassCredential,
			Target:         "server-b.local",
			Params: map[string]string{
				"username": "alice",
				"password": "secret",
			},
		}),
	})
	if err != nil {
		t.Fatalf("AddContext second credential failed: %v", err)
	}

	credentialCmd := &cobra.Command{Use: "credential"}
	outputText := testsupport.CaptureOutput(func() {
		err = ctxcmd.GetCredentialsCmd(credentialCmd, clientHarness.Console)
	})
	if err != nil {
		t.Fatalf("GetCredentialsCmd failed: %v", err)
	}
	if !strings.Contains(outputText, "server-a.local") || !strings.Contains(outputText, "server-b.local") {
		t.Fatalf("credential output missing distinct targets:\n%s", outputText)
	}
}

func TestListContextsHandlesContextWithoutSessionOrTask(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	_, err := clientHarness.Console.Rpc.AddContext(clientHarness.Console.Context(), &clientpb.Context{
		Type: consts.ContextCredential,
		Value: output.MarshalContext(&output.CredentialContext{
			CredentialType: output.UserPassCredential,
			Target:         "host.local",
			Params: map[string]string{
				"username": "alice",
				"password": "secret",
			},
		}),
	})
	if err != nil {
		t.Fatalf("AddContext failed: %v", err)
	}

	contextCmd := &cobra.Command{Use: "context"}
	outputText := testsupport.CaptureOutput(func() {
		err = ctxcmd.ListContexts(contextCmd, clientHarness.Console)
	})
	if err != nil {
		t.Fatalf("ListContexts failed: %v", err)
	}
	if !strings.Contains(outputText, consts.ContextCredential) {
		t.Fatalf("context output missing credential context:\n%s", outputText)
	}
}

func TestSyncCommandIntegrationWritesContextContent(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "ctx-sync-pipe"), true)
	serverSession := h.SeedSession(t, "ctx-sync-session", "ctx-sync-pipe", true)
	task := h.SeedTask(t, serverSession, "download")
	ctxModel := h.SeedDownloadContext(t, task, "sync-me.txt", []byte("sync-body"))
	clientHarness := testsupport.NewClientHarness(t, h)

	oldDir := assets.MaliceDirName
	assets.MaliceDirName = t.TempDir()
	assets.InitLogDir()
	t.Cleanup(func() {
		assets.MaliceDirName = oldDir
		assets.InitLogDir()
	})

	syncCmd := mustNamedCommand(t, ctxcmd.Commands(clientHarness.Console), consts.CommandSync)
	syncCmd.SetArgs([]string{ctxModel.Id})
	if err := syncCmd.Execute(); err != nil {
		t.Fatalf("sync execute failed: %v", err)
	}

	downloadCtx, err := output.ToContext[*output.DownloadContext](ctxModel)
	if err != nil {
		t.Fatalf("ToContext failed: %v", err)
	}
	savePath := filepath.Join(assets.GetTempDir(), ctxModel.Id+"_"+downloadCtx.Name)
	content, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("expected synced file at %s: %v", savePath, err)
	}
	if string(content) != "sync-body" {
		t.Fatalf("synced content = %q, want sync-body", content)
	}
}

func TestGetDownloadsCmdHandlesContextWithoutSession(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	_, err := clientHarness.Console.Rpc.AddContext(clientHarness.Console.Context(), &clientpb.Context{
		Type: consts.ContextDownload,
		Value: output.MarshalContext(&output.DownloadContext{
			FileDescriptor: &output.FileDescriptor{
				Name:       "orphan.bin",
				TargetPath: "C:\\temp\\orphan.bin",
				FilePath:   "C:\\temp\\orphan.bin",
				Size:       10,
			},
		}),
	})
	if err != nil {
		t.Fatalf("AddContext failed: %v", err)
	}

	downloadCmd := &cobra.Command{Use: "download"}
	outputText := testsupport.CaptureOutput(func() {
		err = ctxcmd.GetDownloadsCmd(downloadCmd, clientHarness.Console)
	})
	if err != nil {
		t.Fatalf("GetDownloadsCmd failed: %v", err)
	}
	if !strings.Contains(outputText, "orphan.bin") {
		t.Fatalf("download output missing orphan context:\n%s", outputText)
	}
}

func TestContextDeleteCommandSmokeIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "ctx-delete-pipe"), true)
	serverSession := h.SeedSession(t, "ctx-delete-session", "ctx-delete-pipe", true)
	task := h.SeedTask(t, serverSession, "download")
	ctxModel := h.SeedDownloadContext(t, task, "delete-me.txt", []byte("delete-me"))
	clientHarness := testsupport.NewClientHarness(t, h)

	downloadCtx, err := output.ToContext[*output.DownloadContext](ctxModel)
	if err != nil {
		t.Fatalf("ToContext failed: %v", err)
	}

	root := mustContextRoot(t, ctxcmd.Commands(clientHarness.Console))
	root.SetArgs([]string{"delete", ctxModel.Id, "--yes"})
	if err := root.Execute(); err != nil {
		t.Fatalf("context delete execute failed: %v", err)
	}

	if _, err := h.GetContext(ctxModel.Id); err == nil {
		t.Fatalf("expected deleted context lookup to fail")
	}
	if _, err := os.Stat(downloadCtx.FilePath); !os.IsNotExist(err) {
		t.Fatalf("expected download file to be deleted, stat err=%v", err)
	}
}

func mustContextRoot(t testing.TB, commands []*cobra.Command) *cobra.Command {
	t.Helper()

	for _, cmd := range commands {
		if cmd.Name() == "context" {
			return cmd
		}
	}
	t.Fatalf("context root command not found")
	return nil
}

func mustNamedCommand(t testing.TB, commands []*cobra.Command, name string) *cobra.Command {
	t.Helper()

	for _, cmd := range commands {
		if cmd.Name() == name {
			return cmd
		}
	}
	t.Fatalf("command %s not found", name)
	return nil
}
