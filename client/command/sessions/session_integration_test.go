//go:build integration

package sessions

import (
	"testing"
	"time"

	"github.com/chainreactors/malice-network/server/testsupport"
	"github.com/spf13/cobra"
)

func TestSessionNoteAndGroupIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "session-pipe"), true)
	active := h.SeedSession(t, "sess-active", "session-pipe", true)
	offline := h.SeedSession(t, "sess-offline", "session-pipe", false)
	clientHarness := testsupport.NewClientHarness(t, h)

	note := &cobra.Command{Use: "note"}
	parseSessionArgs(t, note, "active-note", active.ID)
	if err := noteCmd(note, clientHarness.Console); err != nil {
		t.Fatalf("noteCmd(active) failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		sess, ok := clientHarness.Console.Sessions[active.ID]
		return ok && sess.Note == "active-note"
	}, "active session note to update in client state")

	activeModel, err := h.GetSession(active.ID)
	if err != nil {
		t.Fatalf("GetSession(active) failed: %v", err)
	}
	if activeModel.Note != "active-note" {
		t.Fatalf("active session note = %q, want %q", activeModel.Note, "active-note")
	}

	group := &cobra.Command{Use: "group"}
	parseSessionArgs(t, group, "ops", active.ID)
	if err := groupCmd(group, clientHarness.Console); err != nil {
		t.Fatalf("groupCmd(active) failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		sess, ok := clientHarness.Console.Sessions[active.ID]
		return ok && sess.GroupName == "ops"
	}, "active session group to update in client state")

	offlineNote := &cobra.Command{Use: "note"}
	parseSessionArgs(t, offlineNote, "offline-note", offline.ID)
	if err := noteCmd(offlineNote, clientHarness.Console); err != nil {
		t.Fatalf("noteCmd(offline) failed: %v", err)
	}

	offlineGroup := &cobra.Command{Use: "group"}
	parseSessionArgs(t, offlineGroup, "offline-group", offline.ID)
	if err := groupCmd(offlineGroup, clientHarness.Console); err != nil {
		t.Fatalf("groupCmd(offline) failed: %v", err)
	}

	offlineModel, err := h.GetSession(offline.ID)
	if err != nil {
		t.Fatalf("GetSession(offline) failed: %v", err)
	}
	if offlineModel.Note != "offline-note" || offlineModel.GroupName != "offline-group" {
		t.Fatalf("offline session state = note:%q group:%q", offlineModel.Note, offlineModel.GroupName)
	}
}

func TestRemoveSessionIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "remove-pipe"), true)
	active := h.SeedSession(t, "sess-remove", "remove-pipe", true)
	clientHarness := testsupport.NewClientHarness(t, h)

	if _, ok := clientHarness.Console.Sessions[active.ID]; !ok {
		t.Fatalf("expected active session in client cache")
	}

	remove := &cobra.Command{Use: "remove"}
	parseSessionArgs(t, remove, active.ID)
	if err := removeCmd(remove, clientHarness.Console); err != nil {
		t.Fatalf("removeCmd failed: %v", err)
	}

	if _, ok := clientHarness.Console.Sessions[active.ID]; ok {
		t.Fatalf("expected removeCmd to drop session from client cache")
	}

	model, err := h.GetSession(active.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if model != nil {
		t.Fatalf("expected removed session to be hidden from GetSession")
	}
}

func TestRemoveSessionReturnsErrorWhenSessionMissing(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	remove := &cobra.Command{Use: "remove"}
	parseSessionArgs(t, remove, "missing-session")
	if err := removeCmd(remove, clientHarness.Console); err == nil {
		t.Fatal("expected removeCmd to fail for a missing session")
	}
}

func parseSessionArgs(t testing.TB, cmd *cobra.Command, args ...string) {
	t.Helper()

	if err := cmd.Flags().Parse(args); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
}
