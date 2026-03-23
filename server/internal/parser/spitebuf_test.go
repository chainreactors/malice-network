package parser

import (
	"testing"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

func TestSpitesCache_AppendAndBuild(t *testing.T) {
	sc := NewSpitesBuf()

	for i := uint32(0); i < 3; i++ {
		sc.Append(&implantpb.Spite{TaskId: i, Body: &implantpb.Spite_Empty{}})
	}

	if sc.Len() != 3 {
		t.Fatalf("expected Len()=3, got %d", sc.Len())
	}

	spites := sc.Build()
	if len(spites.Spites) != 3 {
		t.Fatalf("expected 3 spites, got %d", len(spites.Spites))
	}

	for i, s := range spites.Spites {
		if s.TaskId != uint32(i) {
			t.Errorf("spite[%d].TaskId = %d, want %d", i, s.TaskId, i)
		}
	}

	// After Build, cache should be empty
	if sc.Len() != 0 {
		t.Fatalf("expected Len()=0 after Build, got %d", sc.Len())
	}
}

func TestSpitesCache_BuildOrEmpty_Empty(t *testing.T) {
	sc := NewSpitesBuf()

	spites := sc.BuildOrEmpty()
	if len(spites.Spites) != 1 {
		t.Fatalf("expected 1 spite (empty body), got %d", len(spites.Spites))
	}

	s := spites.Spites[0]
	if _, ok := s.Body.(*implantpb.Spite_Empty); !ok {
		t.Fatalf("expected Spite_Empty body, got %T", s.Body)
	}
}

func TestSpitesCache_BuildOrEmpty_WithItems(t *testing.T) {
	sc := NewSpitesBuf()
	sc.Append(&implantpb.Spite{TaskId: 10, Body: &implantpb.Spite_Empty{}})
	sc.Append(&implantpb.Spite{TaskId: 20, Body: &implantpb.Spite_Empty{}})

	spites := sc.BuildOrEmpty()
	if len(spites.Spites) != 2 {
		t.Fatalf("expected 2 spites, got %d", len(spites.Spites))
	}
	if spites.Spites[0].TaskId != 10 || spites.Spites[1].TaskId != 20 {
		t.Fatalf("unexpected task IDs: %d, %d", spites.Spites[0].TaskId, spites.Spites[1].TaskId)
	}

	// Cache should be cleared
	if sc.Len() != 0 {
		t.Fatalf("expected Len()=0 after BuildOrEmpty, got %d", sc.Len())
	}
}

func TestSpitesCache_Reset(t *testing.T) {
	sc := NewSpitesBuf()
	sc.Append(&implantpb.Spite{TaskId: 1, Body: &implantpb.Spite_Empty{}})
	sc.Append(&implantpb.Spite{TaskId: 2, Body: &implantpb.Spite_Empty{}})

	if sc.Len() != 2 {
		t.Fatalf("expected Len()=2, got %d", sc.Len())
	}

	sc.Reset()
	if sc.Len() != 0 {
		t.Fatalf("expected Len()=0 after Reset, got %d", sc.Len())
	}
}

func TestSpitesCache_Len(t *testing.T) {
	sc := NewSpitesBuf()

	if sc.Len() != 0 {
		t.Fatalf("expected Len()=0 initially, got %d", sc.Len())
	}

	sc.Append(&implantpb.Spite{Body: &implantpb.Spite_Empty{}})
	if sc.Len() != 1 {
		t.Fatalf("expected Len()=1 after one Append, got %d", sc.Len())
	}

	sc.Append(&implantpb.Spite{Body: &implantpb.Spite_Empty{}})
	if sc.Len() != 2 {
		t.Fatalf("expected Len()=2 after two Appends, got %d", sc.Len())
	}
}

func TestSpitesCache_BuildTwice(t *testing.T) {
	sc := NewSpitesBuf()
	sc.Append(&implantpb.Spite{TaskId: 1, Body: &implantpb.Spite_Empty{}})

	first := sc.Build()
	if len(first.Spites) != 1 {
		t.Fatalf("first Build: expected 1 spite, got %d", len(first.Spites))
	}

	second := sc.Build()
	if len(second.Spites) != 0 {
		t.Fatalf("second Build: expected 0 spites (cache was reset), got %d", len(second.Spites))
	}
}

func TestSpitesCache_AppendNil(t *testing.T) {
	sc := NewSpitesBuf()

	// Appending a nil spite is allowed by the API but could cause NPE downstream
	sc.Append(nil)

	if sc.Len() != 1 {
		t.Fatalf("expected Len()=1 after appending nil, got %d", sc.Len())
	}

	spites := sc.Build()
	if len(spites.Spites) != 1 {
		t.Fatalf("expected 1 spite, got %d", len(spites.Spites))
	}

	// The stored spite is nil, which could cause panics in code that
	// iterates and accesses fields without nil checks.
	if spites.Spites[0] != nil {
		t.Fatal("expected nil spite to be preserved")
	}
}

func TestSpitesCache_BuildOrEmpty_DoesNotResetWhenEmpty(t *testing.T) {
	sc := NewSpitesBuf()

	// BuildOrEmpty on empty cache should NOT call Reset (it is a no-op scenario),
	// and subsequent calls should still return an empty spite.
	first := sc.BuildOrEmpty()
	second := sc.BuildOrEmpty()

	if len(first.Spites) != 1 || len(second.Spites) != 1 {
		t.Fatalf("BuildOrEmpty on empty cache should always return 1 empty spite")
	}
}

func TestSpitesCache_BuildReturnsNewSlice(t *testing.T) {
	sc := NewSpitesBuf()
	sc.Append(&implantpb.Spite{TaskId: 1, Body: &implantpb.Spite_Empty{}})

	built := sc.Build()

	// After build, appending more should not affect the previously built result
	sc.Append(&implantpb.Spite{TaskId: 2, Body: &implantpb.Spite_Empty{}})

	if len(built.Spites) != 1 {
		t.Fatalf("Build result was mutated by subsequent Append")
	}
}
