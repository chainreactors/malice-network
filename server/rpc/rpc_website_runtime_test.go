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
