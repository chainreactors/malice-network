package rpc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

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

func TestListWebContentUsesListenerScopedWebsite(t *testing.T) {
	env := newRPCTestEnv(t)
	_ = env.seedSession(t, "rpc-website-list", "rpc-website-pipe", true)
	server := &Server{}

	for _, item := range []struct {
		listenerID string
		body       string
		path       string
	}{
		{listenerID: "listener-a", body: "a", path: "/a.html"},
		{listenerID: "listener-b", body: "b", path: "/b.html"},
	} {
		if _, err := db.SavePipeline(models.FromPipelinePb(&clientpb.Pipeline{
			Name:       "site-list-shared",
			ListenerId: item.listenerID,
			Type:       consts.WebsitePipeline,
			Body: &clientpb.Pipeline_Web{
				Web: &clientpb.Website{
					Name:       "site-list-shared",
					ListenerId: item.listenerID,
					Root:       "/",
					Port:       8080,
				},
			},
		})); err != nil {
			t.Fatalf("SavePipeline(%s) failed: %v", item.listenerID, err)
		}
		if _, err := db.AddContent(&clientpb.WebContent{
			WebsiteId:  "site-list-shared",
			ListenerId: item.listenerID,
			Path:       item.path,
			Type:       "raw",
			Content:    []byte(item.body),
		}); err != nil {
			t.Fatalf("AddContent(%s) failed: %v", item.listenerID, err)
		}
	}

	contents, err := server.ListWebContent(context.Background(), &clientpb.Website{
		Name:       "site-list-shared",
		ListenerId: "listener-b",
	})
	if err != nil {
		t.Fatalf("ListWebContent failed: %v", err)
	}
	if len(contents.GetContents()) != 1 {
		t.Fatalf("content count = %d, want 1", len(contents.GetContents()))
	}
	content := contents.GetContents()[0]
	if content.GetListenerId() != "listener-b" || content.GetPath() != "/b.html" {
		t.Fatalf("content = listener %q path %q, want listener-b /b.html", content.GetListenerId(), content.GetPath())
	}

	_, err = server.ListWebContent(context.Background(), &clientpb.Website{Name: "site-list-shared"})
	if err == nil || !strings.Contains(err.Error(), "multiple websites named") {
		t.Fatalf("ListWebContent without listener error = %v, want ambiguous website error", err)
	}
}

func TestAddWebsiteContentPersistsRawArtifactPayload(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}

	if _, err := db.SavePipeline(models.FromPipelinePb(&clientpb.Pipeline{
		Name:       "site-artifact",
		ListenerId: "listener-a",
		Type:       consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:       "site-artifact",
				ListenerId: "listener-a",
				Root:       "/",
				Port:       8080,
			},
		},
	})); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)
	content, err := server.AddWebsiteContent(context.Background(), &clientpb.Website{
		Name:       "site-artifact",
		ListenerId: "listener-a",
		Contents: map[string]*clientpb.WebContent{
			"/payload.bin": {
				File:        "beacon-publish",
				Path:        "/payload.bin",
				Type:        "raw",
				Content:     []byte("artifact-payload"),
				ContentType: "application/octet-stream",
				Name:        "payload",
				Comment:     "from artifact",
				Auth:        "none",
			},
		},
	})
	if err != nil {
		t.Fatalf("AddWebsiteContent failed: %v", err)
	}
	if content.GetType() != "raw" || content.GetPath() != "/payload.bin" || content.GetSize() != uint64(len("artifact-payload")) {
		t.Fatalf("content = %#v, want materialized raw artifact content", content)
	}

	stored, err := db.FindWebContent(content.GetId())
	if err != nil {
		t.Fatalf("FindWebContent failed: %v", err)
	}
	storedPb := stored.ToProtobuf(true)
	if string(storedPb.GetContent()) != "artifact-payload" || storedPb.GetName() != "payload" || storedPb.GetComment() != "from artifact" || storedPb.GetAuth() != "none" {
		t.Fatalf("stored content = %#v, want artifact payload with metadata", storedPb)
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlWebContentAdd)
	eventContent := event.Job.GetContents()["/payload.bin"]
	if event.EventType != consts.EventWebsite || eventContent.GetComment() != "from artifact" || eventContent.GetName() != "payload" {
		t.Fatalf("website content event = %#v, want complete committed content", event)
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
	if _, err := server.UpdateWebsiteContentMetadata(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("UpdateWebsiteContentMetadata(nil) error = %v, want %v", err, types.ErrMissingRequestField)
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
	if _, err := server.UpdateWebsiteTLS(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("UpdateWebsiteTLS(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func TestUpdateWebsiteTLSBindsExistingCertificate(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}
	certPEM, keyPEM := websitePEMFixture(t)
	if err := db.SaveCertificate(&models.Certificate{
		Name:    "cert-a",
		Type:    "imported",
		CertPEM: certPEM,
		KeyPEM:  keyPEM,
		Comment: "existing cert",
	}); err != nil {
		t.Fatalf("SaveCertificate failed: %v", err)
	}
	seedWebsitePipelineForTLSTest(t, "site-tls-existing")

	updated, err := server.UpdateWebsiteTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		Name:       "site-tls-existing",
		ListenerId: "listener-a",
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT,
		CertName:   "cert-a",
	})
	if err != nil {
		t.Fatalf("UpdateWebsiteTLS failed: %v", err)
	}
	if updated.CertName != "cert-a" || updated.GetTls() == nil || !updated.GetTls().Enable || updated.GetTls().GetCert().GetComment() != "existing cert" {
		t.Fatalf("updated pipeline = %#v, want bound cert-a with TLS enabled", updated)
	}
}

func TestUpdateWebsiteTLSInlineSaveCreatesCertificateAndBinds(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}
	certPEM, keyPEM := websitePEMFixture(t)
	seedWebsitePipelineForTLSTest(t, "site-tls-save")

	updated, err := server.UpdateWebsiteTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		Name:         "site-tls-save",
		ListenerId:   "listener-a",
		Mode:         clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT,
		Tls:          &clientpb.TLS{Cert: &clientpb.Cert{Cert: certPEM, Key: keyPEM}},
		SaveCert:     true,
		SaveCertName: "site-saved-cert",
		CertComment:  "saved from website",
	})
	if err != nil {
		t.Fatalf("UpdateWebsiteTLS failed: %v", err)
	}
	if updated.CertName != "site-saved-cert" || updated.GetTls() == nil || !updated.GetTls().Enable {
		t.Fatalf("updated pipeline = %#v, want saved cert binding", updated)
	}
	saved, err := db.FindCertificate("site-saved-cert")
	if err != nil {
		t.Fatalf("FindCertificate failed: %v", err)
	}
	if saved.Comment != "saved from website" {
		t.Fatalf("saved comment = %q, want saved from website", saved.Comment)
	}
}

func TestUpdateWebsiteTLSGeneratesTemporarySelfSignedCert(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}
	seedWebsitePipelineForTLSTest(t, "site-tls-generate")

	updated, err := server.UpdateWebsiteTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		Name:       "site-tls-generate",
		ListenerId: "listener-a",
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT,
		Tls:        &clientpb.TLS{Enable: true},
	})
	if err != nil {
		t.Fatalf("UpdateWebsiteTLS failed: %v", err)
	}
	if updated.CertName != "" {
		t.Fatalf("updated cert name = %q, want temporary TLS without cert store binding", updated.CertName)
	}
	if updated.GetTls() == nil || !updated.GetTls().Enable || updated.GetTls().GetCert().GetCert() == "" || updated.GetTls().GetCert().GetKey() == "" {
		t.Fatalf("updated TLS = %#v, want generated certificate material", updated.GetTls())
	}
	certs, err := db.GetAllCertificates()
	if err != nil {
		t.Fatalf("GetAllCertificates failed: %v", err)
	}
	if len(certs) != 0 {
		t.Fatalf("certificate count = %d, want no saved certificate", len(certs))
	}
}

func TestUpdateWebsiteTLSGeneratesAndSavesSelfSignedCert(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}
	seedWebsitePipelineForTLSTest(t, "site-tls-generate-save")

	updated, err := server.UpdateWebsiteTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		Name:         "site-tls-generate-save",
		ListenerId:   "listener-a",
		Mode:         clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT,
		Tls:          &clientpb.TLS{Enable: true},
		SaveCert:     true,
		SaveCertName: "generated-site-cert",
		CertComment:  "generated from website",
	})
	if err != nil {
		t.Fatalf("UpdateWebsiteTLS failed: %v", err)
	}
	if updated.CertName != "generated-site-cert" || updated.GetTls() == nil || !updated.GetTls().Enable {
		t.Fatalf("updated pipeline = %#v, want saved generated cert binding", updated)
	}
	saved, err := db.FindCertificate("generated-site-cert")
	if err != nil {
		t.Fatalf("FindCertificate failed: %v", err)
	}
	if saved.CertPEM == "" || saved.KeyPEM == "" || saved.CACertPEM == "" || saved.CAKeyPEM == "" {
		t.Fatalf("saved cert = %#v, want generated cert and CA material", saved)
	}
	if saved.Comment != "generated from website" {
		t.Fatalf("saved comment = %q, want generated from website", saved.Comment)
	}
}

func TestUpdateWebsiteTLSDisableClearsTLS(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}
	certPEM, keyPEM := websitePEMFixture(t)
	if err := db.SaveCertificate(&models.Certificate{Name: "cert-disable", Type: "imported", CertPEM: certPEM, KeyPEM: keyPEM}); err != nil {
		t.Fatalf("SaveCertificate failed: %v", err)
	}
	seedWebsitePipelineForTLSTest(t, "site-tls-disable")
	model, err := db.FindPipelineByListener("site-tls-disable", "listener-a")
	if err != nil {
		t.Fatalf("FindPipelineByListener failed: %v", err)
	}
	if _, err := db.SetPipelineTLS(model, (&models.Certificate{Name: "cert-disable", Type: "imported", CertPEM: certPEM, KeyPEM: keyPEM}).ToProtobuf(), "cert-disable"); err != nil {
		t.Fatalf("SetPipelineTLS failed: %v", err)
	}

	updated, err := server.UpdateWebsiteTLS(context.Background(), &clientpb.PipelineTLSUpdate{
		Name:       "site-tls-disable",
		ListenerId: "listener-a",
		Mode:       clientpb.TLSUpdateMode_TLS_UPDATE_MODE_DISABLE,
	})
	if err != nil {
		t.Fatalf("UpdateWebsiteTLS failed: %v", err)
	}
	if updated.CertName != "" || updated.GetTls().GetEnable() {
		t.Fatalf("updated pipeline = cert %q tls %#v, want TLS disabled", updated.CertName, updated.GetTls())
	}
	reloaded, err := db.FindPipelineByListener("site-tls-disable", "listener-a")
	if err != nil {
		t.Fatalf("FindPipelineByListener after update failed: %v", err)
	}
	if reloaded.CertName != "" || reloaded.Tls.Enable {
		t.Fatalf("reloaded pipeline = cert %q tls %#v, want TLS disabled", reloaded.CertName, reloaded.Tls)
	}
}

func seedWebsitePipelineForTLSTest(t testing.TB, name string) {
	t.Helper()
	if _, err := db.SavePipeline(models.FromPipelinePb(&clientpb.Pipeline{
		Name:       name,
		ListenerId: "listener-a",
		Type:       consts.WebsitePipeline,
		Tls:        &clientpb.TLS{},
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:       name,
				ListenerId: "listener-a",
				Root:       "/",
				Port:       8080,
			},
		},
	})); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
}

func websitePEMFixture(t testing.TB) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "website.example",
			Organization: []string{"Example Org"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return string(certPEM), string(keyPEM)
}

func TestUpdateWebsiteContentMetadataReturnsUpdatedListFields(t *testing.T) {
	env := newRPCTestEnv(t)
	_ = env.seedSession(t, "rpc-website-metadata", "rpc-website-metadata-pipe", true)
	server := &Server{}

	if _, err := db.SavePipeline(models.FromPipelinePb(&clientpb.Pipeline{
		Name:       "site-metadata",
		ListenerId: "listener-a",
		Type:       consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:       "site-metadata",
				ListenerId: "listener-a",
				Root:       "/",
				Port:       8080,
			},
		},
	})); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	content, err := db.AddContent(&clientpb.WebContent{
		WebsiteId:  "site-metadata",
		ListenerId: "listener-a",
		Path:       "/payload.bin",
		Type:       "raw",
		Content:    []byte("payload"),
	})
	if err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	updated, err := server.UpdateWebsiteContentMetadata(context.Background(), &clientpb.WebContent{
		Id:      content.ID.String(),
		Name:    "payload",
		Comment: "staged content",
	})
	if err != nil {
		t.Fatalf("UpdateWebsiteContentMetadata failed: %v", err)
	}
	if updated.Name != "payload" || updated.Comment != "staged content" {
		t.Fatalf("updated metadata = name %q comment %q, want payload/staged content", updated.Name, updated.Comment)
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlWebContentUpdate)
	if event.EventType != consts.EventWebsite || event.Job.GetPipeline().GetName() != "site-metadata" || event.Job.GetContents()["/payload.bin"].GetComment() != "staged content" {
		t.Fatalf("unexpected website metadata event: %#v", event)
	}
	updated, err = server.UpdateWebsiteContentMetadata(context.Background(), &clientpb.WebContent{
		Id:           content.ID.String(),
		Comment:      "",
		UpdateFields: []string{"comment"},
	})
	if err != nil {
		t.Fatalf("UpdateWebsiteContentMetadata clear comment failed: %v", err)
	}
	if updated.Name != "payload" || updated.Comment != "" {
		t.Fatalf("partial metadata update = name %q comment %q, want payload/empty", updated.Name, updated.Comment)
	}

	contents, err := server.ListWebContent(context.Background(), &clientpb.Website{
		Name:       "site-metadata",
		ListenerId: "listener-a",
	})
	if err != nil {
		t.Fatalf("ListWebContent failed: %v", err)
	}
	if len(contents.GetContents()) != 1 {
		t.Fatalf("content count = %d, want 1", len(contents.GetContents()))
	}
	got := contents.GetContents()[0]
	if got.Name != "payload" || got.Comment != "" {
		t.Fatalf("listed metadata = name %q comment %q, want payload/empty", got.Name, got.Comment)
	}
}

func TestUpdateWebsiteContentUsesSingleRuntimeUpdateEvent(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}
	listener := core.NewListener("listener-content-update", "127.0.0.1")
	core.Listeners.Add(listener)
	pipeline := &clientpb.Pipeline{
		Name:       "site-content-update",
		ListenerId: listener.Name,
		Type:       consts.WebsitePipeline,
		Enable:     true,
		Body: &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name:       "site-content-update",
			ListenerId: listener.Name,
			Root:       "/",
			Port:       8080,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	content, err := db.AddContent(&clientpb.WebContent{
		WebsiteId:  pipeline.Name,
		ListenerId: listener.Name,
		Path:       "/payload.bin",
		Type:       "raw",
		Content:    []byte("old"),
	})
	if err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}
	core.Jobs.AddPipeline(pipeline)
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	type updateResult struct {
		content *clientpb.WebContent
		err     error
	}
	result := make(chan updateResult, 1)
	go func() {
		updated, updateErr := server.UpdateWebsiteContent(context.Background(), &clientpb.WebContent{
			Id:      content.ID.String(),
			Comment: "updated",
			Content: []byte("new"),
		})
		result <- updateResult{content: updated, err: updateErr}
	}()

	var ctrl *clientpb.JobCtrl
	select {
	case ctrl = <-listener.Ctrl:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for website content control")
	}
	if ctrl.Ctrl != consts.CtrlWebContentUpdate {
		t.Fatalf("control operation = %q, want %q", ctrl.Ctrl, consts.CtrlWebContentUpdate)
	}
	listener.CtrlJob.Store(ctrl.Id, nil)
	if got := ctrl.Job.GetPipeline().GetWeb().GetContents()["/payload.bin"].GetComment(); got != "updated" {
		t.Fatalf("event content comment = %q, want updated", got)
	}
	if got := string(ctrl.Content.GetContent()); got != "new" {
		t.Fatalf("runtime content = %q, want new", got)
	}
	select {
	case got := <-result:
		t.Fatalf("UpdateWebsiteContent returned before runtime acknowledgement: content=%#v err=%v", got.content, got.err)
	case <-time.After(50 * time.Millisecond):
	}

	assertNoLifecycleEvent(t, events, consts.CtrlWebContentUpdate, 100*time.Millisecond)

	handleJobStatus(listener, &clientpb.JobStatus{
		CtrlId: ctrl.Id,
		Ctrl:   ctrl.Ctrl,
		Status: consts.CtrlStatusSuccess,
		Job:    ctrl.Job,
	})
	select {
	case got := <-result:
		if got.err != nil {
			t.Fatalf("UpdateWebsiteContent failed: %v", got.err)
		}
		if got.content.GetComment() != "updated" {
			t.Fatalf("updated comment = %q, want updated", got.content.GetComment())
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for UpdateWebsiteContent result")
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlWebContentUpdate)
	if event.EventType != consts.EventWebsite || event.Job.GetContents()["/payload.bin"].GetComment() != "updated" {
		t.Fatalf("website update event = %#v, want complete committed content", event)
	}
}

func TestRemoveWebsiteContentDeletesBeforeRuntimeStatus(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}
	listener := core.NewListener("listener-content-remove", "127.0.0.1")
	core.Listeners.Add(listener)
	pipeline := &clientpb.Pipeline{
		Name:       "site-content-remove",
		ListenerId: listener.Name,
		Type:       consts.WebsitePipeline,
		Enable:     true,
		Body: &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name:       "site-content-remove",
			ListenerId: listener.Name,
			Root:       "/",
			Port:       8080,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	content, err := db.AddContent(&clientpb.WebContent{
		WebsiteId:  pipeline.Name,
		ListenerId: listener.Name,
		Path:       "/payload.bin",
		Type:       "raw",
		Content:    []byte("old"),
	})
	if err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}
	core.Jobs.AddPipeline(pipeline)
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	result := make(chan error, 1)
	go func() {
		_, removeErr := server.RemoveWebsiteContent(context.Background(), &clientpb.WebContent{Id: content.ID.String()})
		result <- removeErr
	}()

	var ctrl *clientpb.JobCtrl
	select {
	case ctrl = <-listener.Ctrl:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for website content remove control")
	}
	if ctrl.Ctrl != consts.CtrlWebContentRemove {
		t.Fatalf("control operation = %q, want %q", ctrl.Ctrl, consts.CtrlWebContentRemove)
	}
	listener.CtrlJob.Store(ctrl.Id, nil)
	if _, err := db.FindWebContent(content.ID.String()); err == nil {
		t.Fatal("content should be deleted before the runtime success event is published")
	}
	select {
	case err := <-result:
		t.Fatalf("RemoveWebsiteContent returned before runtime acknowledgement: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	handleJobStatus(listener, &clientpb.JobStatus{
		CtrlId: ctrl.Id,
		Ctrl:   ctrl.Ctrl,
		Status: consts.CtrlStatusSuccess,
		Job:    ctrl.Job,
	})
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("RemoveWebsiteContent failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for RemoveWebsiteContent result")
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlWebContentRemove)
	if event.EventType != consts.EventWebsite || event.Job.GetContents()["/payload.bin"].GetPath() != "/payload.bin" {
		t.Fatalf("website remove event = %#v, want removed path", event)
	}
}

func TestRemoveWebsiteContentUsesPersistedIdentity(t *testing.T) {
	newRPCTestEnv(t)
	pipeline := &clientpb.Pipeline{
		Name:       "site-content-remove-identity",
		ListenerId: "listener-content-remove-identity",
		Type:       consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name:       "site-content-remove-identity",
			ListenerId: "listener-content-remove-identity",
			Root:       "/",
			Port:       8080,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	content, err := db.AddContent(&clientpb.WebContent{
		WebsiteId:  pipeline.Name,
		ListenerId: pipeline.ListenerId,
		Path:       "/payload.bin",
		Type:       "raw",
		Content:    []byte("payload"),
	})
	if err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	if _, err := (&Server{}).RemoveWebsiteContent(context.Background(), &clientpb.WebContent{
		Id:         content.ID.String(),
		WebsiteId:  "stale-website",
		ListenerId: "stale-listener",
		Path:       "/stale.bin",
	}); err != nil {
		t.Fatalf("RemoveWebsiteContent should use persisted identity: %v", err)
	}
	if _, err := db.FindWebContent(content.ID.String()); err == nil {
		t.Fatal("persisted content was not removed")
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlWebContentRemove)
	if got := event.Job.GetPipeline(); got.GetName() != pipeline.Name || got.GetListenerId() != pipeline.ListenerId {
		t.Fatalf("remove event pipeline = %#v, want persisted website identity", got)
	}
	if got := event.Job.GetContents()["/payload.bin"].GetPath(); got != "/payload.bin" {
		t.Fatalf("remove event path = %q, want persisted path", got)
	}
}

func TestRemoveWebsiteContentRollsBackWhenControlCannotQueue(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}
	listener := core.NewListener("listener-content-remove-rollback", "127.0.0.1")
	core.Listeners.Add(listener)
	pipeline := &clientpb.Pipeline{
		Name:       "site-content-remove-rollback",
		ListenerId: listener.Name,
		Type:       consts.WebsitePipeline,
		Enable:     true,
		Body: &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name:       "site-content-remove-rollback",
			ListenerId: listener.Name,
			Root:       "/",
			Port:       8080,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	content, err := db.AddContent(&clientpb.WebContent{
		WebsiteId:  pipeline.Name,
		ListenerId: listener.Name,
		Path:       "/payload.bin",
		Type:       "raw",
		Content:    []byte("old"),
	})
	if err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}
	core.Jobs.AddPipeline(pipeline)
	for index := 0; index < cap(listener.Ctrl); index++ {
		listener.Ctrl <- &clientpb.JobCtrl{Ctrl: consts.CtrlPipelineSync}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = server.RemoveWebsiteContent(ctx, &clientpb.WebContent{Id: content.ID.String()})
	if err == nil {
		t.Fatal("RemoveWebsiteContent should fail when runtime control cannot be queued")
	}
	stored, findErr := db.FindWebContent(content.ID.String())
	if findErr != nil {
		t.Fatalf("removed content was not restored: %v", findErr)
	}
	if got := string(stored.ToProtobuf(true).GetContent()); got != "old" {
		t.Fatalf("restored content = %q, want old", got)
	}
}

func TestUpdateWebsiteContentRollsBackWhenControlCannotQueue(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}
	listener := core.NewListener("listener-content-rollback", "127.0.0.1")
	core.Listeners.Add(listener)
	pipeline := &clientpb.Pipeline{
		Name:       "site-content-rollback",
		ListenerId: listener.Name,
		Type:       consts.WebsitePipeline,
		Enable:     true,
		Body: &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name:       "site-content-rollback",
			ListenerId: listener.Name,
			Root:       "/",
			Port:       8080,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	content, err := db.AddContent(&clientpb.WebContent{
		WebsiteId:  pipeline.Name,
		ListenerId: listener.Name,
		Path:       "/payload.bin",
		Type:       "raw",
		Comment:    "old",
		Content:    []byte("old"),
	})
	if err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}
	core.Jobs.AddPipeline(pipeline)
	for index := 0; index < cap(listener.Ctrl); index++ {
		listener.Ctrl <- &clientpb.JobCtrl{Ctrl: consts.CtrlPipelineSync}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = server.UpdateWebsiteContent(ctx, &clientpb.WebContent{
		Id:      content.ID.String(),
		Comment: "updated",
		Content: []byte("new"),
	})
	if err == nil {
		t.Fatal("UpdateWebsiteContent should fail when runtime control cannot be queued")
	}
	stored, findErr := db.FindWebContent(content.ID.String())
	if findErr != nil {
		t.Fatalf("FindWebContent failed: %v", findErr)
	}
	storedPB := stored.ToProtobuf(true)
	if storedPB.GetComment() != "old" || string(storedPB.GetContent()) != "old" {
		t.Fatalf("stored content = comment %q body %q, want rolled back old values", storedPB.GetComment(), storedPB.GetContent())
	}
}

func TestAddWebsiteContentRestoresExistingContentWhenControlCannotQueue(t *testing.T) {
	newRPCTestEnv(t)
	server := &Server{}
	listener := core.NewListener("listener-content-add-rollback", "127.0.0.1")
	core.Listeners.Add(listener)
	pipeline := &clientpb.Pipeline{
		Name:       "site-content-add-rollback",
		ListenerId: listener.Name,
		Type:       consts.WebsitePipeline,
		Enable:     true,
		Body: &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name:       "site-content-add-rollback",
			ListenerId: listener.Name,
			Root:       "/",
			Port:       8080,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	existing, err := db.AddContent(&clientpb.WebContent{
		WebsiteId:  pipeline.Name,
		ListenerId: listener.Name,
		Path:       "/payload.bin",
		Type:       "raw",
		Comment:    "old",
		Content:    []byte("old"),
	})
	if err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}
	core.Jobs.AddPipeline(pipeline)
	for index := 0; index < cap(listener.Ctrl); index++ {
		listener.Ctrl <- &clientpb.JobCtrl{Ctrl: consts.CtrlPipelineSync}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = server.AddWebsiteContent(ctx, &clientpb.Website{
		Name:       pipeline.Name,
		ListenerId: listener.Name,
		Contents: map[string]*clientpb.WebContent{
			"/payload.bin": {
				Path:    "/payload.bin",
				Type:    "raw",
				Comment: "new",
				Content: []byte("new"),
			},
		},
	})
	if err == nil {
		t.Fatal("AddWebsiteContent should fail when runtime control cannot be queued")
	}
	stored, findErr := db.FindWebContent(existing.ID.String())
	if findErr != nil {
		t.Fatalf("overwritten content was not restored: %v", findErr)
	}
	storedPB := stored.ToProtobuf(true)
	if storedPB.GetComment() != "old" || string(storedPB.GetContent()) != "old" {
		t.Fatalf("restored content = comment %q body %q, want old values", storedPB.GetComment(), storedPB.GetContent())
	}
}

func TestRegisterWebsiteRejectsColonName(t *testing.T) {
	newRPCTestEnv(t)
	core.Listeners.Add(core.NewListener("listener-web-colon", "127.0.0.1"))

	_, err := (&Server{}).RegisterWebsite(context.Background(), &clientpb.Pipeline{
		Name:       "web:bad",
		ListenerId: "listener-web-colon",
		Type:       consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:       "web:bad",
				ListenerId: "listener-web-colon",
			},
		},
	})
	if err == nil {
		t.Fatal("RegisterWebsite should reject ':' in website pipeline name")
	}
}

func TestStopWebsiteDisablesPipelineWhenListenerIsOffline(t *testing.T) {
	newRPCTestEnv(t)
	pipeline := &clientpb.Pipeline{
		Name:       "site-offline-stop",
		ListenerId: "listener-site-offline-stop",
		Type:       consts.WebsitePipeline,
		Enable:     true,
		Body: &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name:       "site-offline-stop",
			ListenerId: "listener-site-offline-stop",
			Root:       "/",
			Port:       8080,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	core.Jobs.AddPipeline(pipeline)
	listed, err := (&Server{}).ListWebsites(context.Background(), &clientpb.Listener{})
	if err != nil {
		t.Fatalf("ListWebsites failed: %v", err)
	}
	if len(listed.Pipelines) != 1 || listed.Pipelines[0].Enable {
		t.Fatalf("offline website list = %#v, want one inactive pipeline", listed.Pipelines)
	}
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	if _, err := (&Server{}).StopWebsite(context.Background(), &clientpb.CtrlPipeline{
		Name:       pipeline.Name,
		ListenerId: pipeline.ListenerId,
	}); err != nil {
		t.Fatalf("StopWebsite failed for offline listener: %v", err)
	}
	stored, err := db.FindPipelineByListener(pipeline.Name, pipeline.ListenerId)
	if err != nil {
		t.Fatalf("FindPipelineByListener failed: %v", err)
	}
	if stored.Enable {
		t.Fatal("StopWebsite should disable the persisted pipeline")
	}
	if _, err := core.Jobs.GetByListener(pipeline.Name, pipeline.ListenerId); err == nil {
		t.Fatal("StopWebsite should remove the stale runtime job")
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlWebsiteStop)
	if event.EventType != consts.EventWebsite || event.Job.GetPipeline().GetName() != pipeline.Name {
		t.Fatalf("unexpected offline website stop event: %#v", event)
	}
}

func TestDeleteWebsiteRemovesPipelineWhenListenerIsOffline(t *testing.T) {
	newRPCTestEnv(t)
	pipeline := &clientpb.Pipeline{
		Name:       "site-offline-delete",
		ListenerId: "listener-site-offline-delete",
		Type:       consts.WebsitePipeline,
		Enable:     true,
		Body: &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name:       "site-offline-delete",
			ListenerId: "listener-site-offline-delete",
			Root:       "/",
			Port:       8080,
		}},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	if _, err := (&Server{}).DeleteWebsite(context.Background(), &clientpb.CtrlPipeline{
		Name:       pipeline.Name,
		ListenerId: pipeline.ListenerId,
	}); err != nil {
		t.Fatalf("DeleteWebsite failed for offline listener: %v", err)
	}
	if _, err := db.FindPipelineByListener(pipeline.Name, pipeline.ListenerId); err == nil {
		t.Fatal("DeleteWebsite should remove the persisted pipeline")
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlWebsiteDelete)
	if event.EventType != consts.EventWebsite || event.Job.GetPipeline().GetName() != pipeline.Name {
		t.Fatalf("unexpected offline website delete event: %#v", event)
	}
}
