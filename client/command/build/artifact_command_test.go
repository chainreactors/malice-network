package build_test

import (
	"context"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestArtifactCommentCommandForwardsUpdateArtifact(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	if err := h.ExecuteClient(consts.CommandArtifact, "comment", "demo-artifact", "new comment"); err != nil {
		t.Fatalf("artifact comment failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*clientpb.Artifact](t, h, "UpdateArtifact")
	if req.Name != "demo-artifact" {
		t.Fatalf("artifact name = %q, want demo-artifact", req.Name)
	}
	if req.Comment != "new comment" {
		t.Fatalf("comment = %q, want new comment", req.Comment)
	}
	testsupport.RequireNoSessionEvents(t, h)
}

func TestArtifactInspectDownloadsArtifact(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	if err := h.ExecuteClient(consts.CommandArtifact, "inspect", "demo-artifact"); err != nil {
		t.Fatalf("artifact inspect failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*clientpb.Artifact](t, h, "DownloadArtifact")
	if req.Name != "demo-artifact" {
		t.Fatalf("artifact name = %q, want demo-artifact", req.Name)
	}
}

func TestArtifactPublishAddsWebsiteContent(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	if err := h.ExecuteClient(consts.CommandArtifact, "publish", "demo-artifact", "--website", "site-a", "--path", "/payload.bin"); err != nil {
		t.Fatalf("artifact publish failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 2 || calls[0].Method != "DownloadArtifact" || calls[1].Method != "AddWebsiteContent" {
		t.Fatalf("calls = %#v", calls)
	}
	download := calls[0].Request.(*clientpb.Artifact)
	if download.Name != "demo-artifact" {
		t.Fatalf("download artifact request = %#v", download)
	}
	add := calls[1].Request.(*clientpb.Website)
	content := add.Contents["/payload.bin"]
	if add.Name != "site-a" || content == nil || string(content.Content) != "artifact-bin" {
		t.Fatalf("add website content request = %#v", add)
	}
}

func TestArtifactPruneDeletesFailedArtifacts(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Recorder.OnArtifacts("ListArtifact", func(context.Context, any) (*clientpb.Artifacts, error) {
		return &clientpb.Artifacts{Artifacts: []*clientpb.Artifact{
			{Name: "failed", Status: "failed", CreatedAt: time.Now().Unix()},
			{Name: "done", Status: consts.BuildStatusCompleted, CreatedAt: time.Now().Unix()},
		}}, nil
	})

	if err := h.ExecuteClient(consts.CommandArtifact, "prune", "--failed"); err != nil {
		t.Fatalf("artifact prune failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 2 || calls[0].Method != "ListArtifact" || calls[1].Method != "DeleteArtifact" {
		t.Fatalf("calls = %#v", calls)
	}
	req := calls[1].Request.(*clientpb.Artifact)
	if req.Name != "failed" {
		t.Fatalf("deleted artifact = %q, want failed", req.Name)
	}
}
