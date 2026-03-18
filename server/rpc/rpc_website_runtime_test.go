package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func TestMapContentsRejectsNonWebsitePipeline(t *testing.T) {
	err := MapContents(nil)
	if err == nil || !strings.Contains(err.Error(), "website pipeline required") {
		t.Fatalf("MapContents(nil) error = %v, want website pipeline required", err)
	}

	err = MapContents(&clientpb.Pipeline{Name: "tcp-a", Type: consts.TCPPipeline})
	if err == nil || !strings.Contains(err.Error(), "website pipeline required") {
		t.Fatalf("MapContents(non-website) error = %v, want website pipeline required", err)
	}
}

func TestCloneWebsiteJobDoesNotMutateOriginalContents(t *testing.T) {
	original := &core.Job{
		ID:   7,
		Name: "site-a",
		Pipeline: &clientpb.Pipeline{
			Name: "site-a",
			Type: consts.WebsitePipeline,
			Body: &clientpb.Pipeline_Web{
				Web: &clientpb.Website{
					Name: "site-a",
					Contents: map[string]*clientpb.WebContent{
						"/old.html": {Path: "/old.html"},
					},
				},
			},
		},
	}

	cloned := cloneWebsiteJob(original, map[string]*clientpb.WebContent{
		"/new.html": {Path: "/new.html"},
	})
	if cloned == nil || cloned.GetPipeline() == nil || cloned.GetPipeline().GetWeb() == nil {
		t.Fatalf("cloneWebsiteJob returned invalid job: %#v", cloned)
	}
	if _, ok := cloned.GetPipeline().GetWeb().Contents["/new.html"]; !ok {
		t.Fatalf("cloned contents = %#v, want new content entry", cloned.GetPipeline().GetWeb().Contents)
	}
	if _, ok := original.Pipeline.GetWeb().Contents["/old.html"]; !ok {
		t.Fatalf("original contents mutated: %#v", original.Pipeline.GetWeb().Contents)
	}
	if _, ok := original.Pipeline.GetWeb().Contents["/new.html"]; ok {
		t.Fatalf("original contents should not gain cloned entry: %#v", original.Pipeline.GetWeb().Contents)
	}
}

func TestMapContentsInitializesNilContentsMap(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-website-map", "rpc-website-pipe", true)
	_ = sess

	listener, err := core.Listeners.Get("test-listener")
	if err != nil {
		t.Fatalf("listener lookup failed: %v", err)
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(&clientpb.Pipeline{
		Name:       "site-map-nil",
		ListenerId: listener.Name,
		Type:       consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name: "site-map-nil",
				Root: "/",
				Port: 8080,
			},
		},
	})); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	if _, err := db.AddContent(&clientpb.WebContent{
		WebsiteId: "site-map-nil",
		Path:      "/index.html",
		Type:      "raw",
		Content:   []byte("hello"),
	}); err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}

	pipeline := &clientpb.Pipeline{
		Name: "site-map-nil",
		Type: consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:     "site-map-nil",
				Contents: nil,
			},
		},
	}

	if err := MapContents(pipeline); err != nil {
		t.Fatalf("MapContents failed: %v", err)
	}
	if pipeline.GetWeb().Contents == nil {
		t.Fatal("MapContents should initialize contents map")
	}
	if _, ok := pipeline.GetWeb().Contents["/index.html"]; !ok {
		t.Fatalf("contents = %#v, want /index.html", pipeline.GetWeb().Contents)
	}
}

func TestWebsiteHandlersRejectNilRequest(t *testing.T) {
	server := &Server{}

	if _, err := server.ListWebContent(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("ListWebContent(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.AddWebsiteContent(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("AddWebsiteContent(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.UpdateWebsiteContent(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("UpdateWebsiteContent(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.RemoveWebsiteContent(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("RemoveWebsiteContent(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.RegisterWebsite(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("RegisterWebsite(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.RegisterWebsite(context.Background(), &clientpb.Pipeline{Name: "web-a"}); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("RegisterWebsite(non-web) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.StartWebsite(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("StartWebsite(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}
