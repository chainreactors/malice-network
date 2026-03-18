package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
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
