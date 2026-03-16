package context_test

import (
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	ctxcmd "github.com/chainreactors/malice-network/client/command/context"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	clientcore "github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
)

func TestAddDownloadRequiresSession(t *testing.T) {
	con := &clientcore.Console{Log: iomclient.Log}

	if _, err := ctxcmd.AddDownload(con, nil, &clientpb.Task{}, &output.FileDescriptor{}); err == nil {
		t.Fatal("expected AddDownload to fail when session is nil")
	}
}

func TestAddDownloadRequiresTask(t *testing.T) {
	con := &clientcore.Console{Log: iomclient.Log}
	sess := &iomclient.Session{
		Session: &clientpb.Session{SessionId: "sess-1"},
	}

	if _, err := ctxcmd.AddDownload(con, sess, nil, &output.FileDescriptor{}); err == nil {
		t.Fatal("expected AddDownload to fail when task is nil")
	}
}

func TestContextAddHelpersRequireSessionAndTask(t *testing.T) {
	type addFunc func(*clientcore.Console, *iomclient.Session, *clientpb.Task) (bool, error)

	cases := []struct {
		name string
		run  addFunc
	}{
		{
			name: "credential",
			run: func(con *clientcore.Console, sess *iomclient.Session, task *clientpb.Task) (bool, error) {
				return ctxcmd.AddCredential(con, sess, task, output.UserPassCredential, map[string]string{"username": "alice"})
			},
		},
		{
			name: "port",
			run: func(con *clientcore.Console, sess *iomclient.Session, task *clientpb.Task) (bool, error) {
				return ctxcmd.AddPort(con, sess, task, []*output.Port{{Port: "80", Protocol: "tcp"}})
			},
		},
		{
			name: "keylogger",
			run: func(con *clientcore.Console, sess *iomclient.Session, task *clientpb.Task) (bool, error) {
				return ctxcmd.AddKeylogger(con, sess, task, []byte("typed"))
			},
		},
		{
			name: "upload",
			run: func(con *clientcore.Console, sess *iomclient.Session, task *clientpb.Task) (bool, error) {
				return ctxcmd.AddUpload(con, sess, task, &output.FileDescriptor{Name: "upload.bin"})
			},
		},
		{
			name: "screenshot",
			run: func(con *clientcore.Console, sess *iomclient.Session, task *clientpb.Task) (bool, error) {
				return ctxcmd.AddScreenshot(con, sess, task, []byte("shot"))
			},
		},
	}

	con := &clientcore.Console{Log: iomclient.Log}
	validSession := &iomclient.Session{Session: &clientpb.Session{SessionId: "sess-1"}}
	validTask := &clientpb.Task{SessionId: "sess-1", TaskId: 7}

	for _, tc := range cases {
		t.Run(tc.name+"_requires_session", func(t *testing.T) {
			if _, err := tc.run(con, nil, validTask); err == nil {
				t.Fatalf("%s should fail when session is nil", tc.name)
			}
		})
		t.Run(tc.name+"_requires_task", func(t *testing.T) {
			if _, err := tc.run(con, validSession, nil); err == nil {
				t.Fatalf("%s should fail when task is nil", tc.name)
			}
		})
	}
}

func TestAddScreenshotUsesContentPayload(t *testing.T) {
	h := testsupport.NewHarness(t)

	ok, err := ctxcmd.AddScreenshot(h.Console, h.Session, &clientpb.Task{
		SessionId: h.Session.SessionId,
		TaskId:    9,
	}, []byte("image-bytes"))
	if err != nil {
		t.Fatalf("AddScreenshot failed: %v", err)
	}
	if !ok {
		t.Fatal("AddScreenshot should report success")
	}

	req, _ := testsupport.MustSingleCall[*clientpb.Context](t, h, "AddScreenShot")
	if string(req.Content) != "image-bytes" {
		t.Fatalf("screenshot content = %q, want image-bytes", req.Content)
	}
	if len(req.Value) != 0 {
		t.Fatalf("screenshot value should be empty when raw content is sent, got %q", req.Value)
	}
}
