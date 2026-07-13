package core

import (
	"fmt"
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/client"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

func TestSessionAddonAccessorsCloneDeduplicateAndRefresh(t *testing.T) {
	sess := &Session{}
	seatbelt := &implantpb.Addon{Name: "seatbelt", Type: "bof", Depend: "exec"}
	sess.ReplaceAddons([]*implantpb.Addon{nil, seatbelt, {Name: "seatbelt", Type: "assembly"}, {Name: ""}})
	seatbelt.Type = "mutated-after-replace"

	sess.MergeAddons([]*implantpb.Addon{
		{Name: "seatbelt", Type: "assembly", Depend: "execute_dll"},
		{Name: "sharpview", Type: "assembly", Depend: "exec"},
	})
	addons := sess.AddonsSnapshot()
	if len(addons) != 2 {
		t.Fatalf("addon count = %d, want 2: %#v", len(addons), addons)
	}
	if addons[0].GetName() != "seatbelt" || addons[0].GetType() != "assembly" || addons[0].GetDepend() != "execute_dll" {
		t.Fatalf("seatbelt addon = %#v, want refreshed clone", addons[0])
	}
	if !sess.HasAddon("sharpview") || sess.HasAddon("missing") {
		t.Fatalf("HasAddon results unexpected: addons=%#v", addons)
	}
	addons[0].Type = "mutated-snapshot"
	if got := sess.AddonsSnapshot()[0].GetType(); got != "assembly" {
		t.Fatalf("stored addon type = %q after snapshot mutation, want assembly", got)
	}
}

func TestSessionTimerAndRoutingSnapshotsDoNotTear(t *testing.T) {
	sess := &Session{}
	const iterations = 2000
	var writers sync.WaitGroup
	writers.Add(1)
	go func() {
		defer writers.Done()
		for i := 0; i < iterations; i++ {
			suffix := i % 2
			sess.SetTimer(fmt.Sprintf("expr-%d", suffix), float64(suffix))
			sess.stateMu.Lock()
			sess.Target = fmt.Sprintf("target-%d", suffix)
			sess.ListenerID = fmt.Sprintf("listener-%d", suffix)
			sess.PipelineID = fmt.Sprintf("pipeline-%d", suffix)
			sess.stateMu.Unlock()
		}
	}()

	for i := 0; i < iterations; i++ {
		expression, jitter := sess.TimerSnapshot()
		if expression != "" && expression != fmt.Sprintf("expr-%d", int(jitter)) {
			t.Fatalf("torn timer snapshot = %q/%v", expression, jitter)
		}
		target, listenerID, pipelineID := sess.ConnectionSnapshot()
		if listenerID != "" && (target[len(target)-1:] != listenerID[len(listenerID)-1:] || listenerID[len(listenerID)-1:] != pipelineID[len(pipelineID)-1:]) {
			t.Fatalf("torn connection snapshot = %q/%q/%q", target, listenerID, pipelineID)
		}
	}
	writers.Wait()
}

func TestSessionTimerAccessorsInitializeMissingSessionInfo(t *testing.T) {
	for _, sess := range []*Session{
		{},
		{SessionContext: &client.SessionContext{}},
	} {
		sess.SetTimer("*/5 * * * *", 0.25)
		expression, jitter := sess.TimerSnapshot()
		if expression != "*/5 * * * *" || jitter != 0.25 {
			t.Fatalf("timer snapshot = %q/%v, want initialized values", expression, jitter)
		}
	}
}
