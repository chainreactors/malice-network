package db

import (
	"errors"
	"testing"

	"github.com/chainreactors/malice-network/server/internal/db/models"
)

const (
	linkSessionA = "11111111-1111-4111-8111-111111111111"
	linkSessionB = "22222222-2222-4222-8222-222222222222"
	linkSessionC = "33333333-3333-4333-8333-333333333333"
)

func TestSetSessionLinkCreatesAndReparents(t *testing.T) {
	initTestDB(t)
	seedSessionLinkEndpoints(t, linkSessionA, linkSessionB, linkSessionC)

	created, err := SetSessionLink(linkSessionA, linkSessionC, models.SessionLinkSourceManual)
	if err != nil {
		t.Fatalf("SetSessionLink create failed: %v", err)
	}
	if created.ParentSessionID != linkSessionA || created.ChildSessionID != linkSessionC {
		t.Fatalf("created link = %#v", created)
	}
	if created.Source != models.SessionLinkSourceManual {
		t.Fatalf("source = %q, want %q", created.Source, models.SessionLinkSourceManual)
	}

	reparented, err := SetSessionLink(linkSessionB, linkSessionC, models.SessionLinkSourceManual)
	if err != nil {
		t.Fatalf("SetSessionLink reparent failed: %v", err)
	}
	if reparented.ParentSessionID != linkSessionB {
		t.Fatalf("parent = %q, want %q", reparented.ParentSessionID, linkSessionB)
	}

	links, err := ListSessionLinks("", linkSessionC)
	if err != nil {
		t.Fatalf("ListSessionLinks failed: %v", err)
	}
	if len(links) != 1 || links[0].ParentSessionID != linkSessionB {
		t.Fatalf("links = %#v, want one link through %s", links, linkSessionB)
	}
	oldParentLinks, err := ListSessionLinks(linkSessionA, "")
	if err != nil {
		t.Fatalf("ListSessionLinks old parent failed: %v", err)
	}
	if len(oldParentLinks) != 0 {
		t.Fatalf("old parent links = %#v, want none", oldParentLinks)
	}
}

func TestSetSessionLinkRejectsInvalidTopology(t *testing.T) {
	initTestDB(t)
	seedSessionLinkEndpoints(t, linkSessionA, linkSessionB, linkSessionC)

	if _, err := SetSessionLink(linkSessionA, linkSessionA, models.SessionLinkSourceManual); !errors.Is(err, ErrSessionLinkSelf) {
		t.Fatalf("self link error = %v, want %v", err, ErrSessionLinkSelf)
	}
	if _, err := SetSessionLink(linkSessionA, "44444444-4444-4444-8444-444444444444", models.SessionLinkSourceManual); !errors.Is(err, ErrSessionLinkSessionNotFound) {
		t.Fatalf("missing session error = %v, want %v", err, ErrSessionLinkSessionNotFound)
	}

	if _, err := SetSessionLink(linkSessionA, linkSessionB, models.SessionLinkSourceManual); err != nil {
		t.Fatalf("SetSessionLink A -> B failed: %v", err)
	}
	if _, err := SetSessionLink(linkSessionB, linkSessionC, models.SessionLinkSourceManual); err != nil {
		t.Fatalf("SetSessionLink B -> C failed: %v", err)
	}
	if _, err := SetSessionLink(linkSessionC, linkSessionA, models.SessionLinkSourceManual); !errors.Is(err, ErrSessionLinkCycle) {
		t.Fatalf("cycle error = %v, want %v", err, ErrSessionLinkCycle)
	}
}

func TestRemoveSessionLinkAndSessionCleanup(t *testing.T) {
	initTestDB(t)
	seedSessionLinkEndpoints(t, linkSessionA, linkSessionB, linkSessionC)

	if _, err := SetSessionLink(linkSessionA, linkSessionB, models.SessionLinkSourceManual); err != nil {
		t.Fatalf("SetSessionLink A -> B failed: %v", err)
	}
	if _, err := SetSessionLink(linkSessionB, linkSessionC, models.SessionLinkSourceManual); err != nil {
		t.Fatalf("SetSessionLink B -> C failed: %v", err)
	}
	if err := RemoveSessionLink(linkSessionB); err != nil {
		t.Fatalf("RemoveSessionLink failed: %v", err)
	}
	if err := RemoveSessionLink(linkSessionB); !errors.Is(err, ErrSessionLinkNotFound) {
		t.Fatalf("second RemoveSessionLink error = %v, want %v", err, ErrSessionLinkNotFound)
	}

	if _, err := SetSessionLink(linkSessionA, linkSessionB, models.SessionLinkSourceManual); err != nil {
		t.Fatalf("restore A -> B failed: %v", err)
	}
	if err := RemoveSession(linkSessionB); err != nil {
		t.Fatalf("RemoveSession failed: %v", err)
	}
	links, err := ListSessionLinks("", "")
	if err != nil {
		t.Fatalf("ListSessionLinks failed: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("links after removing middle session = %#v, want none", links)
	}
}

func seedSessionLinkEndpoints(t *testing.T, sessionIDs ...string) {
	t.Helper()
	for _, sessionID := range sessionIDs {
		if err := Client.Create(&models.Session{SessionID: sessionID}).Error; err != nil {
			t.Fatalf("create session %s: %v", sessionID, err)
		}
	}
}
