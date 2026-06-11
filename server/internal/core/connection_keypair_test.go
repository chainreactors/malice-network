package core

import (
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
)

// TestGetKeyPairForSession_Disabled verifies that nil is returned when
// secure is not enabled, regardless of key content.
func TestGetKeyPairForSession_Disabled(t *testing.T) {
	t.Parallel()
	cfg := &implanttypes.SecureConfig{Enable: false}
	kp := GetKeyPairForSession(1, cfg)
	if kp != nil {
		t.Fatal("expected nil KeyPair when secure is disabled")
	}
}

// TestGetKeyPairForSession_NilConfig verifies nil config returns nil KeyPair.
func TestGetKeyPairForSession_NilConfig(t *testing.T) {
	t.Parallel()
	kp := GetKeyPairForSession(1, nil)
	if kp != nil {
		t.Fatal("expected nil KeyPair for nil config")
	}
}

// TestGetKeyPairForSession_ColdStart verifies that an empty (non-nil) KeyPair
// is returned when secure is enabled but no pre-shared keys exist.
// This is the cold-start scenario where encryption starts after key exchange.
func TestGetKeyPairForSession_ColdStart(t *testing.T) {
	t.Parallel()
	cfg := &implanttypes.SecureConfig{
		Enable:           true,
		ServerPublicKey:  "",
		ServerPrivateKey: "",
		ImplantPublicKey: "",
	}
	kp := GetKeyPairForSession(42, cfg)
	if kp == nil {
		t.Fatal("expected non-nil KeyPair for cold-start secure config")
	}
	if kp.PublicKey != "" {
		t.Errorf("PublicKey = %q, want empty", kp.PublicKey)
	}
	if kp.PrivateKey != "" {
		t.Errorf("PrivateKey = %q, want empty", kp.PrivateKey)
	}
}

// TestGetKeyPairForSession_WithPreSharedKeys verifies that pre-shared keys
// are correctly assembled into the KeyPair.
func TestGetKeyPairForSession_WithPreSharedKeys(t *testing.T) {
	t.Parallel()
	cfg := &implanttypes.SecureConfig{
		Enable:           true,
		ServerPrivateKey: "server-priv",
		ImplantPublicKey: "implant-pub",
	}
	kp := GetKeyPairForSession(99, cfg)
	if kp == nil {
		t.Fatal("expected non-nil KeyPair")
	}
	if kp.PublicKey != "implant-pub" {
		t.Errorf("PublicKey = %q, want %q", kp.PublicKey, "implant-pub")
	}
	if kp.PrivateKey != "server-priv" {
		t.Errorf("PrivateKey = %q, want %q", kp.PrivateKey, "server-priv")
	}
}

// TestGetKeyPairForSession_AfterKeyExchange verifies that session-level keys
// (from key exchange) take priority over pipeline-level pre-shared keys.
func TestGetKeyPairForSession_AfterKeyExchange(t *testing.T) {
	t.Parallel()
	var sid uint32 = 200
	cfg := &implanttypes.SecureConfig{
		Enable:           true,
		ServerPrivateKey: "pipeline-server-priv",
		ImplantPublicKey: "pipeline-implant-pub",
	}

	// Simulate key exchange result stored in ListenerSessions
	ListenerSessions.Add(&clientpb.Session{
		RawId: sid,
		KeyPair: &clientpb.KeyPair{
			PublicKey:  "exchanged-pub",
			PrivateKey: "exchanged-priv",
		},
	})
	t.Cleanup(func() { ListenerSessions.Remove(sid) })

	kp := GetKeyPairForSession(sid, cfg)
	if kp == nil {
		t.Fatal("expected non-nil KeyPair")
	}
	if kp.PublicKey != "exchanged-pub" {
		t.Errorf("PublicKey = %q, want %q", kp.PublicKey, "exchanged-pub")
	}
	// PrivateKey should contain both exchanged and pipeline keys
	if kp.PrivateKey == "" {
		t.Fatal("PrivateKey should not be empty")
	}
}
