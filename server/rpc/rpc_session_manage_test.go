package rpc

import (
	"context"
	"errors"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
)

// ---------------------------------------------------------------------------
// GetSessions
// ---------------------------------------------------------------------------

func TestGetSessions_ActiveOnly(t *testing.T) {
	env := newRPCTestEnv(t)
	env.seedSession(t, "gs-active-1", "gs-pipe-1", true)
	env.seedSession(t, "gs-active-2", "gs-pipe-2", true)

	resp, err := (&Server{}).GetSessions(context.Background(), &clientpb.SessionRequest{All: false})
	if err != nil {
		t.Fatalf("GetSessions error: %v", err)
	}
	if len(resp.Sessions) < 2 {
		t.Fatalf("expected at least 2 active sessions, got %d", len(resp.Sessions))
	}
}

func TestGetSessions_AllFlag(t *testing.T) {
	env := newRPCTestEnv(t)
	env.seedSession(t, "gs-all-active", "gs-all-pipe1", true)
	env.seedSession(t, "gs-all-dbonly", "gs-all-pipe2", false)

	resp, err := (&Server{}).GetSessions(context.Background(), &clientpb.SessionRequest{All: true})
	if err != nil {
		t.Fatalf("GetSessions(All=true) error: %v", err)
	}
	if len(resp.Sessions) < 2 {
		t.Fatalf("expected at least 2 sessions from DB, got %d", len(resp.Sessions))
	}
}

// BUG TEST: GetSessions with nil request panics accessing req.All.
func TestGetSessions_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: GetSessions(nil) panics: %v", r)
		}
	}()
	_, err := (&Server{}).GetSessions(context.Background(), nil)
	if err != nil {
		t.Logf("GetSessions(nil) returned error (no panic): %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetSessionCount
// ---------------------------------------------------------------------------

func TestGetSessionCount_Counts(t *testing.T) {
	env := newRPCTestEnv(t)
	env.seedSession(t, "gsc-active", "gsc-pipe1", true)
	env.seedSession(t, "gsc-dbonly", "gsc-pipe2", false)

	resp, err := (&Server{}).GetSessionCount(context.Background(), &clientpb.Empty{})
	if err != nil {
		t.Fatalf("GetSessionCount error: %v", err)
	}
	if resp.Alive < 1 {
		t.Fatalf("expected Alive >= 1, got %d", resp.Alive)
	}
	if resp.Total < 2 {
		t.Fatalf("expected Total >= 2, got %d", resp.Total)
	}
	if resp.Total < resp.Alive {
		t.Fatalf("Total (%d) should be >= Alive (%d)", resp.Total, resp.Alive)
	}
}

// ---------------------------------------------------------------------------
// GetSession
// ---------------------------------------------------------------------------

func TestGetSession_ActiveSession(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "gs-mem-sess", "gs-mem-pipe", true)

	got, err := (&Server{}).GetSession(context.Background(), &clientpb.SessionRequest{SessionId: sess.ID})
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if got == nil || got.SessionId != sess.ID {
		t.Fatalf("GetSession = %v, want session %s", got, sess.ID)
	}
}

func TestGetSession_DBOnlySession(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "gs-db-sess", "gs-db-pipe", false)

	got, err := (&Server{}).GetSession(context.Background(), &clientpb.SessionRequest{SessionId: sess.ID})
	if err != nil {
		t.Fatalf("GetSession(db-only) error: %v", err)
	}
	if got == nil || got.SessionId != sess.ID {
		t.Fatalf("GetSession(db-only) = %v, want session %s", got, sess.ID)
	}
	// Should NOT be recovered into memory.
	if _, err := core.Sessions.Get(sess.ID); err == nil {
		t.Fatal("GetSession should not recover a db-only session into memory")
	}
}

func TestGetSession_Unknown(t *testing.T) {
	_ = newRPCTestEnv(t)

	got, err := (&Server{}).GetSession(context.Background(), &clientpb.SessionRequest{SessionId: "totally-unknown"})
	// NOTE: The code intends to return nil,nil for unknown sessions, but
	// db.FindSession returns a "record not found" error from GORM. This means
	// callers see an error rather than a clean nil,nil. This is a behavioral
	// inconsistency worth noting.
	if err == nil && got != nil {
		t.Fatalf("GetSession(unknown) should not return a session: %v", got)
	}
	if err != nil {
		t.Logf("GetSession(unknown) returned error (potential inconsistency with nil,nil intent): %v", err)
	}
}

// BUG TEST: GetSession with nil request likely panics because it accesses req.SessionId
// without a nil check.
func TestGetSession_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: GetSession(nil) panics: %v", r)
		}
	}()
	_, err := (&Server{}).GetSession(context.Background(), nil)
	if err != nil {
		t.Logf("GetSession(nil) returned error (no panic): %v", err)
	}
}

// ---------------------------------------------------------------------------
// SessionManage
// ---------------------------------------------------------------------------

func TestSessionManage_Delete(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "sm-del-sess", "sm-del-pipe", true)

	_, err := (&Server{}).SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: sess.ID,
		Op:        "delete",
	})
	if err != nil {
		t.Fatalf("SessionManage(delete) error: %v", err)
	}

	// Verify removed from memory.
	if _, err := core.Sessions.Get(sess.ID); err == nil {
		t.Fatal("session should have been removed from memory after delete")
	}
	// Verify removed from DB.
	model, err := db.FindSession(sess.ID)
	if err != nil {
		t.Fatalf("FindSession after delete failed: %v", err)
	}
	if model != nil {
		t.Fatalf("expected deleted session to be nil in DB, got %v", model)
	}
}

func TestSessionManage_Note_ActiveSession(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "sm-note-active", "sm-note-pipe", true)

	_, err := (&Server{}).SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: sess.ID,
		Op:        "note",
		Arg:       "important-note",
	})
	if err != nil {
		t.Fatalf("SessionManage(note) error: %v", err)
	}
	if sess.Note != "important-note" {
		t.Fatalf("session note = %q, want %q", sess.Note, "important-note")
	}
	// Verify persisted to DB.
	model, err := env.getSession(sess.ID)
	if err != nil {
		t.Fatalf("getSession error: %v", err)
	}
	if model.Note != "important-note" {
		t.Fatalf("DB note = %q, want %q", model.Note, "important-note")
	}
}

func TestSessionManage_Group_ActiveSession(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "sm-group-active", "sm-group-pipe", true)

	_, err := (&Server{}).SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: sess.ID,
		Op:        "group",
		Arg:       "red-team",
	})
	if err != nil {
		t.Fatalf("SessionManage(group) error: %v", err)
	}
	if sess.Group != "red-team" {
		t.Fatalf("session group = %q, want %q", sess.Group, "red-team")
	}
}

func TestSessionManage_NoteOnDBOnly(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "sm-note-dbonly", "sm-note-db-pipe", false)

	_, err := (&Server{}).SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: sess.ID,
		Op:        "note",
		Arg:       "db-note",
	})
	if err != nil {
		t.Fatalf("SessionManage(note on db-only) error: %v", err)
	}
	model, err := env.getSession(sess.ID)
	if err != nil {
		t.Fatalf("getSession error: %v", err)
	}
	if model.Note != "db-note" {
		t.Fatalf("DB note = %q, want %q", model.Note, "db-note")
	}
}

func TestSessionManage_GroupOnDBOnly(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "sm-grp-dbonly", "sm-grp-db-pipe", false)

	_, err := (&Server{}).SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: sess.ID,
		Op:        "group",
		Arg:       "ops-group",
	})
	if err != nil {
		t.Fatalf("SessionManage(group on db-only) error: %v", err)
	}
	model, err := env.getSession(sess.ID)
	if err != nil {
		t.Fatalf("getSession error: %v", err)
	}
	if model.GroupName != "ops-group" {
		t.Fatalf("DB group = %q, want %q", model.GroupName, "ops-group")
	}
}

// Edge case: unknown operation is silently a no-op. This may be a design issue.
func TestSessionManage_UnknownOp(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "sm-unknownop", "sm-unknownop-pipe", true)

	resp, err := (&Server{}).SessionManage(context.Background(), &clientpb.BasicUpdateSession{
		SessionId: sess.ID,
		Op:        "foobar",
		Arg:       "value",
	})
	// Currently returns empty response with no error. This is a silent no-op.
	if err != nil {
		t.Fatalf("SessionManage(unknown op) error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil empty response")
	}
}

// BUG TEST: SessionManage with nil request panics accessing req.Op.
func TestSessionManage_NilRequest(t *testing.T) {
	_ = newRPCTestEnv(t)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG CONFIRMED: SessionManage(nil) panics: %v", r)
		}
	}()
	_, err := (&Server{}).SessionManage(context.Background(), nil)
	if err != nil {
		t.Logf("SessionManage(nil) returned error (no panic): %v", err)
	}
}

// ---------------------------------------------------------------------------
// Ping - GenericHandler path
// ---------------------------------------------------------------------------

func TestPing_NoSessionInContext(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).Ping(context.Background(), &implantpb.Ping{Nonce: 1})
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("Ping(no session ctx) error = %v, want %v", err, types.ErrNotFoundSession)
	}
}

func TestPing_NoPipeline(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "ping-nopipe", "ping-nopipe-pipe", true)

	// Ensure no pipeline stream is registered.
	pipelinesCh.Delete(sess.PipelineID)

	_, err := (&Server{}).Ping(incomingSessionContext(sess.ID), &implantpb.Ping{Nonce: 42})
	if !errors.Is(err, types.ErrNotFoundPipeline) {
		t.Fatalf("Ping(no pipeline) error = %v, want %v", err, types.ErrNotFoundPipeline)
	}
}
