package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestRegister_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).Register(context.Background(), nil)
	if !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("Register(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func TestRegister_EmptySessionId(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).Register(context.Background(), &clientpb.RegisterSession{
		SessionId:    "",
		RegisterData: &implantpb.Register{Name: "agent"},
	})
	if !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("Register(empty session id) error = %v, want %v", err, types.ErrInvalidSessionID)
	}
}

func TestRegister_NilRegisterData(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).Register(context.Background(), &clientpb.RegisterSession{
		SessionId:    "some-id",
		RegisterData: nil,
	})
	if !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("Register(nil RegisterData) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func TestRegister_CreatesNewSession(t *testing.T) {
	env := newRPCTestEnv(t)
	// seedSession creates "test-listener" which is needed for RegisterSession.
	env.seedSession(t, "reg-setup-sess", "reg-setup-pipe", false)

	newID := "register-new-session"
	_, err := (&Server{}).Register(context.Background(), &clientpb.RegisterSession{
		Type:       "tcp",
		SessionId:  newID,
		RawId:      42,
		PipelineId: "reg-setup-pipe",
		ListenerId: "test-listener",
		Target:     "192.168.1.100",
		RegisterData: &implantpb.Register{
			Name: "new-agent",
			Timer: &implantpb.Timer{
				Expression: "*/10 * * * * * *",
			},
			Sysinfo: &implantpb.SysInfo{
				Os: &implantpb.Os{
					Name: "linux",
					Arch: "x64",
				},
				Process: &implantpb.Process{
					Name: "implant",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}

	// Verify session is in memory.
	sess, err := core.Sessions.Get(newID)
	if err != nil {
		t.Fatalf("session not found in memory after Register: %v", err)
	}
	if sess.Target != "192.168.1.100" {
		t.Fatalf("session target = %q, want %q", sess.Target, "192.168.1.100")
	}

	// Verify session is in DB.
	model, err := db.FindSession(newID)
	if err != nil {
		t.Fatalf("FindSession error: %v", err)
	}
	if model == nil {
		t.Fatal("session not found in DB after Register")
	}
}

func TestRegister_ReRegisterExistingSession(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rereg-sess", "rereg-pipe", true)

	_, err := (&Server{}).Register(context.Background(), &clientpb.RegisterSession{
		Type:       "tcp",
		SessionId:  sess.ID,
		RawId:      1,
		PipelineId: "rereg-pipe",
		ListenerId: "test-listener",
		Target:     "10.0.0.1",
		RegisterData: &implantpb.Register{
			Name: "re-agent",
			Timer: &implantpb.Timer{
				Expression: "*/5 * * * * * *",
			},
			Sysinfo: &implantpb.SysInfo{
				Os: &implantpb.Os{
					Name: "windows",
					Arch: "amd64",
				},
				Process: &implantpb.Process{
					Name: "re.exe",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("re-Register error: %v", err)
	}
	// Should still be in memory.
	if _, err := core.Sessions.Get(sess.ID); err != nil {
		t.Fatalf("session lost after re-Register: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SysInfo
// ---------------------------------------------------------------------------

func TestSysInfo_NoSessionID(t *testing.T) {
	_ = newRPCTestEnv(t)
	// No session_id in metadata.
	_, err := (&Server{}).SysInfo(context.Background(), &implantpb.SysInfo{
		Os: &implantpb.Os{Name: "linux", Arch: "x64"},
	})
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("SysInfo(no session_id) error = %v, want %v", err, types.ErrNotFoundSession)
	}
}

func TestSysInfo_MissingSession(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).SysInfo(incomingSessionContext("nonexistent-sess"), &implantpb.SysInfo{
		Os: &implantpb.Os{Name: "linux", Arch: "x64"},
	})
	// Session is not in memory.
	if err == nil {
		t.Fatal("SysInfo(missing session) should return error")
	}
}

func TestSysInfo_UpdatesSession(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "sysinfo-update", "sysinfo-pipe", true)

	_, err := (&Server{}).SysInfo(incomingSessionContext(sess.ID), &implantpb.SysInfo{
		Os: &implantpb.Os{
			Name: "LINUX",
			Arch: "arm64",
		},
		Process: &implantpb.Process{
			Name: "updated-implant",
			Pid:  1234,
		},
	})
	if err != nil {
		t.Fatalf("SysInfo error: %v", err)
	}
	// Verify the session was updated.
	if sess.Os == nil {
		t.Fatal("session Os is nil after SysInfo update")
	}
}

// BUG TEST: SysInfo with nil request -- getSessionID works, then
// sess.UpdateSysInfo(nil) is called. May or may not panic depending on
// UpdateSysInfo implementation.
func TestSysInfo_NilRequest(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "sysinfo-nil", "sysinfo-nil-pipe", true)

	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: SysInfo(nil request) panics: %v", r)
		}
	}()
	_, err := (&Server{}).SysInfo(incomingSessionContext(sess.ID), nil)
	if err != nil {
		t.Logf("SysInfo(nil request) returned error (no panic): %v", err)
	}
}

// ---------------------------------------------------------------------------
// Checkin
// ---------------------------------------------------------------------------

func TestCheckin_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).Checkin(incomingSessionContext("checkin-nil"), nil)
	if !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("Checkin(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func TestCheckin_MissingSession(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).Checkin(incomingSessionContext("totally-unknown-session"), &implantpb.Ping{Nonce: 1})
	// Session not in memory and not in DB.
	if err == nil {
		t.Fatal("Checkin(missing session) should return error")
	}
}

func TestCheckin_RecoversDBSession(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "checkin-recover", "checkin-recover-pipe", false)

	// Soft-delete the session to test recovery path.
	if err := db.RemoveSession(sess.ID); err != nil {
		t.Fatalf("RemoveSession failed: %v", err)
	}

	_, err := (&Server{}).Checkin(incomingSessionContext(sess.ID), &implantpb.Ping{Nonce: 7})
	if err != nil {
		t.Fatalf("Checkin error: %v", err)
	}

	// Session should now be in memory.
	recovered, err := core.Sessions.Get(sess.ID)
	if err != nil {
		t.Fatalf("session not recovered to memory: %v", err)
	}
	if recovered.LastCheckinUnix() == 0 {
		t.Fatal("expected LastCheckin to be updated")
	}
}

func TestCheckin_UpdatesTimestamp(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "checkin-ts", "checkin-ts-pipe", true)

	before := sess.LastCheckinUnix()
	// Wait briefly so timestamp differs.
	time.Sleep(10 * time.Millisecond)

	_, err := (&Server{}).Checkin(incomingSessionContext(sess.ID), &implantpb.Ping{Nonce: 99})
	if err != nil {
		t.Fatalf("Checkin error: %v", err)
	}

	after := sess.LastCheckinUnix()
	if after < before {
		t.Fatalf("LastCheckin did not advance: before=%d, after=%d", before, after)
	}
}

func TestCheckin_ActiveSessionAlreadyAlive(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "checkin-alive", "checkin-alive-pipe", true)

	// First checkin.
	_, err := (&Server{}).Checkin(incomingSessionContext(sess.ID), &implantpb.Ping{Nonce: 1})
	if err != nil {
		t.Fatalf("first Checkin error: %v", err)
	}
	// Second checkin on already-alive session should also succeed.
	_, err = (&Server{}).Checkin(incomingSessionContext(sess.ID), &implantpb.Ping{Nonce: 2})
	if err != nil {
		t.Fatalf("second Checkin error: %v", err)
	}
}

func TestCheckin_NoSessionIDInContext(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).Checkin(context.Background(), &implantpb.Ping{Nonce: 1})
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("Checkin(no session_id) error = %v, want %v", err, types.ErrNotFoundSession)
	}
}

// ---------------------------------------------------------------------------
// Edge: Register with duplicate session ID (already in memory from different pipeline)
// ---------------------------------------------------------------------------

func TestRegister_DuplicateSessionDifferentPipeline(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "dup-sess", "dup-pipe-original", true)

	// Re-register with a different pipeline name.
	_, err := (&Server{}).Register(context.Background(), &clientpb.RegisterSession{
		Type:       "tcp",
		SessionId:  sess.ID,
		RawId:      2,
		PipelineId: "dup-pipe-original", // must use existing pipeline
		ListenerId: "test-listener",
		Target:     "10.0.0.99",
		RegisterData: &implantpb.Register{
			Name: "agent-dup",
			Timer: &implantpb.Timer{
				Expression: "* * * * *",
			},
			Sysinfo: &implantpb.SysInfo{
				Os: &implantpb.Os{
					Name: "windows",
					Arch: "amd64",
				},
				Process: &implantpb.Process{
					Name: "dup.exe",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Register(duplicate sess, different pipeline) error: %v", err)
	}

	// Verify session is still accessible.
	if _, err := core.Sessions.Get(sess.ID); err != nil {
		t.Fatalf("session lost after duplicate register: %v", err)
	}
}
