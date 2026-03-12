package db

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

// newTestPipeline creates a simple TCP pipeline model for testing.
func newTestPipeline(name, listenerID string) *models.Pipeline {
	return &models.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Type:       consts.TCPPipeline,
		Host:       "127.0.0.1",
		Port:       8443,
		PipelineParams: &implanttypes.PipelineParams{
			Tls:        &implanttypes.TlsConfig{},
			Encryption: implanttypes.EncryptionsConfig{},
		},
	}
}

// ============================================
// Pipeline CRUD Tests
// ============================================

func TestSavePipeline_Create(t *testing.T) {
	initTestDB(t)

	p := newTestPipeline("sp-create-1", "ls-1")
	saved, err := SavePipeline(p)
	if err != nil {
		t.Fatalf("SavePipeline (create) failed: %v", err)
	}
	if saved.Name != "sp-create-1" {
		t.Errorf("expected name 'sp-create-1', got %q", saved.Name)
	}
}

func TestSavePipeline_Update(t *testing.T) {
	initTestDB(t)

	p := newTestPipeline("sp-update-1", "ls-1")
	SavePipeline(p)

	// Update same pipeline
	p2 := newTestPipeline("sp-update-1", "ls-1")
	p2.Host = "10.0.0.1"
	updated, err := SavePipeline(p2)
	if err != nil {
		t.Fatalf("SavePipeline (update) failed: %v", err)
	}
	if updated.Host != "10.0.0.1" {
		t.Errorf("expected host '10.0.0.1', got %q", updated.Host)
	}
}

func TestSavePipeline_Nil(t *testing.T) {
	initTestDB(t)

	_, err := SavePipeline(nil)
	if err == nil {
		t.Error("SavePipeline(nil) should return error")
	}
}

func TestFindPipeline(t *testing.T) {
	initTestDB(t)

	p := newTestPipeline("fp-find-1", "ls-1")
	SavePipeline(p)

	found, err := FindPipeline("fp-find-1")
	if err != nil {
		t.Fatalf("FindPipeline failed: %v", err)
	}
	if found.Name != "fp-find-1" {
		t.Errorf("expected name 'fp-find-1', got %q", found.Name)
	}
}

func TestFindPipeline_NotFound(t *testing.T) {
	initTestDB(t)

	_, err := FindPipeline("nonexistent")
	if err == nil {
		t.Error("FindPipeline should return error for nonexistent pipeline")
	}
}

func TestFindPipelineByListener(t *testing.T) {
	initTestDB(t)

	SavePipeline(newTestPipeline("fpl-1", "ls-a"))
	SavePipeline(newTestPipeline("fpl-2", "ls-b"))

	found, err := FindPipelineByListener("fpl-1", "ls-a")
	if err != nil {
		t.Fatalf("FindPipelineByListener failed: %v", err)
	}
	if found.ListenerId != "ls-a" {
		t.Errorf("expected listener 'ls-a', got %q", found.ListenerId)
	}

	// Wrong listener
	_, err = FindPipelineByListener("fpl-1", "ls-b")
	if err == nil {
		t.Error("FindPipelineByListener should return error for mismatched listener")
	}
}

func TestDeletePipeline(t *testing.T) {
	initTestDB(t)

	SavePipeline(newTestPipeline("dp-1", "ls-1"))

	if err := DeletePipeline("dp-1"); err != nil {
		t.Fatalf("DeletePipeline failed: %v", err)
	}

	_, err := FindPipeline("dp-1")
	if err == nil {
		t.Error("expected error after deleting pipeline")
	}
}

func TestDeletePipelineByListener(t *testing.T) {
	initTestDB(t)

	SavePipeline(newTestPipeline("dpl-1", "ls-x"))

	if err := DeletePipelineByListener("dpl-1", "ls-x"); err != nil {
		t.Fatalf("DeletePipelineByListener failed: %v", err)
	}

	_, err := FindPipeline("dpl-1")
	if err == nil {
		t.Error("expected error after deleting pipeline by listener")
	}
}

func TestEnableDisablePipeline(t *testing.T) {
	initTestDB(t)

	p := newTestPipeline("ed-1", "ls-1")
	p.Enable = false
	SavePipeline(p)

	// Enable
	if err := EnablePipeline("ed-1"); err != nil {
		t.Fatalf("EnablePipeline failed: %v", err)
	}
	found, _ := FindPipeline("ed-1")
	if !found.Enable {
		t.Error("expected pipeline to be enabled")
	}

	// Disable
	if err := DisablePipeline("ed-1"); err != nil {
		t.Fatalf("DisablePipeline failed: %v", err)
	}
	found, _ = FindPipeline("ed-1")
	if found.Enable {
		t.Error("expected pipeline to be disabled")
	}
}

func TestEnableDisablePipelineByListener(t *testing.T) {
	initTestDB(t)

	p := newTestPipeline("edl-1", "ls-1")
	p.Enable = false
	SavePipeline(p)

	if err := EnablePipelineByListener("edl-1", "ls-1"); err != nil {
		t.Fatalf("EnablePipelineByListener failed: %v", err)
	}
	found, _ := FindPipelineByListener("edl-1", "ls-1")
	if !found.Enable {
		t.Error("expected pipeline to be enabled")
	}

	if err := DisablePipelineByListener("edl-1", "ls-1"); err != nil {
		t.Fatalf("DisablePipelineByListener failed: %v", err)
	}
	found, _ = FindPipelineByListener("edl-1", "ls-1")
	if found.Enable {
		t.Error("expected pipeline to be disabled")
	}
}

// ============================================
// Certificate CRUD Tests
// ============================================

func TestSaveCertificate(t *testing.T) {
	initTestDB(t)

	cert := &models.Certificate{
		Name:    "cert-1",
		Type:    "selfsigned",
		CertPEM: "test-cert-pem",
		KeyPEM:  "test-key-pem",
	}
	if err := SaveCertificate(cert); err != nil {
		t.Fatalf("SaveCertificate failed: %v", err)
	}
}

func TestSaveCertificate_Duplicate(t *testing.T) {
	initTestDB(t)

	cert := &models.Certificate{Name: "cert-dup", Type: "selfsigned"}
	SaveCertificate(cert)

	err := SaveCertificate(&models.Certificate{Name: "cert-dup", Type: "selfsigned"})
	if err == nil {
		t.Error("SaveCertificate should return error for duplicate name")
	}
}

func TestFindCertificate(t *testing.T) {
	initTestDB(t)

	SaveCertificate(&models.Certificate{Name: "cert-find-1", Type: "selfsigned"})

	found, err := FindCertificate("cert-find-1")
	if err != nil {
		t.Fatalf("FindCertificate failed: %v", err)
	}
	if found.Name != "cert-find-1" {
		t.Errorf("expected name 'cert-find-1', got %q", found.Name)
	}
}

func TestFindCertificate_NotFound(t *testing.T) {
	initTestDB(t)

	_, err := FindCertificate("nonexistent")
	if err == nil {
		t.Error("FindCertificate should return error for nonexistent cert")
	}
}

func TestDeleteCertificate(t *testing.T) {
	initTestDB(t)

	SaveCertificate(&models.Certificate{Name: "cert-del-1", Type: "selfsigned"})

	if err := DeleteCertificate("cert-del-1"); err != nil {
		t.Fatalf("DeleteCertificate failed: %v", err)
	}

	_, err := FindCertificate("cert-del-1")
	if err == nil {
		t.Error("expected error after deleting certificate")
	}
}

func TestDeleteCertificate_NotFound(t *testing.T) {
	initTestDB(t)

	// Should not error when deleting nonexistent cert
	if err := DeleteCertificate("nonexistent"); err != nil {
		t.Fatalf("DeleteCertificate should not error for nonexistent cert, got: %v", err)
	}
}

func TestGetAllCertificates(t *testing.T) {
	initTestDB(t)

	SaveCertificate(&models.Certificate{Name: "cert-ga-1", Type: "selfsigned"})
	SaveCertificate(&models.Certificate{Name: "cert-ga-2", Type: "imported"})

	certs, err := GetAllCertificates()
	if err != nil {
		t.Fatalf("GetAllCertificates failed: %v", err)
	}
	if len(certs) != 2 {
		t.Errorf("expected 2 certificates, got %d", len(certs))
	}
}

func TestUpdateCert(t *testing.T) {
	initTestDB(t)

	SaveCertificate(&models.Certificate{
		Name:    "cert-upd-1",
		Type:    "selfsigned",
		CertPEM: "old-cert",
		KeyPEM:  "old-key",
	})

	if err := UpdateCert("cert-upd-1", "new-cert", "new-key", "new-ca"); err != nil {
		t.Fatalf("UpdateCert failed: %v", err)
	}

	found, _ := FindCertificate("cert-upd-1")
	if found.CertPEM != "new-cert" {
		t.Errorf("expected CertPEM 'new-cert', got %q", found.CertPEM)
	}
	if found.KeyPEM != "new-key" {
		t.Errorf("expected KeyPEM 'new-key', got %q", found.KeyPEM)
	}
	if found.CACertPEM != "new-ca" {
		t.Errorf("expected CACertPEM 'new-ca', got %q", found.CACertPEM)
	}
}

func TestIsDuplicateCommonNameAndCAType(t *testing.T) {
	initTestDB(t)

	if isDuplicateCommonNameAndCAType("noexist") {
		t.Error("should return false for nonexistent name")
	}

	SaveCertificate(&models.Certificate{Name: "dup-check", Type: "selfsigned"})

	if !isDuplicateCommonNameAndCAType("dup-check") {
		t.Error("should return true for existing name")
	}
}

// ============================================
// FindPipelineCert Test
// ============================================

func TestFindPipelineCert(t *testing.T) {
	initTestDB(t)

	// Pipeline without cert
	SavePipeline(newTestPipeline("fpc-1", "ls-1"))

	cert, err := FindPipelineCert("fpc-1", "ls-1")
	if err != nil {
		t.Fatalf("FindPipelineCert failed: %v", err)
	}
	if cert != nil {
		t.Error("expected nil cert for pipeline without CertName")
	}
}
