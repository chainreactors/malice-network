package core

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

func withIsolatedTicker(t *testing.T) {
	t.Helper()

	oldTicker := GlobalTicker
	GlobalTicker = NewTicker()
	t.Cleanup(func() {
		GlobalTicker.RemoveAll()
		GlobalTicker = oldTicker
	})
}

func TestCacheGetMessagesReturnsOrderedTaskStream(t *testing.T) {
	withIsolatedTicker(t)

	cache := NewCache(filepath.Join(t.TempDir(), "cache.bin"))
	cache.duration = time.Hour

	cache.AddMessage(&implantpb.Spite{TaskId: 41, Name: "second"}, 1)
	cache.AddMessage(&implantpb.Spite{TaskId: 41, Name: "third"}, 2)
	cache.AddMessage(&implantpb.Spite{TaskId: 41, Name: "first"}, 0)
	cache.AddMessage(&implantpb.Spite{TaskId: 99, Name: "other"}, 0)

	messages, ok := cache.GetMessages(41)
	if !ok {
		t.Fatal("GetMessages should return the stored task stream")
	}
	if len(messages) != 3 {
		t.Fatalf("GetMessages length = %d, want 3", len(messages))
	}
	if messages[0].Name != "first" || messages[1].Name != "second" || messages[2].Name != "third" {
		t.Fatalf("GetMessages order = %q, %q, %q", messages[0].Name, messages[1].Name, messages[2].Name)
	}

	last, ok := cache.GetLastMessage(41)
	if !ok {
		t.Fatal("GetLastMessage should return the latest task message")
	}
	if last.Name != "third" {
		t.Fatalf("GetLastMessage name = %q, want third", last.Name)
	}
}

func TestCacheTrimDropsExpiredEntriesBeforeOldestLiveEntries(t *testing.T) {
	cache := &Cache{
		maxSize: 2,
	}
	now := time.Now().Unix()
	cache.items.Store("expired", &clientpb.SpiteCacheItem{
		Id:         "expired",
		Index:      0,
		Spite:      &implantpb.Spite{TaskId: 1, Name: "expired"},
		Expiration: now - 1,
	})
	cache.items.Store("oldest-live", &clientpb.SpiteCacheItem{
		Id:         "oldest-live",
		Index:      0,
		Spite:      &implantpb.Spite{TaskId: 1, Name: "oldest-live"},
		Expiration: now + 10,
	})
	cache.items.Store("newer-live", &clientpb.SpiteCacheItem{
		Id:         "newer-live",
		Index:      1,
		Spite:      &implantpb.Spite{TaskId: 1, Name: "newer-live"},
		Expiration: now + 20,
	})
	cache.items.Store("newest-live", &clientpb.SpiteCacheItem{
		Id:         "newest-live",
		Index:      2,
		Spite:      &implantpb.Spite{TaskId: 1, Name: "newest-live"},
		Expiration: now + 30,
	})

	cache.trim()

	if _, ok := cache.items.Load("expired"); ok {
		t.Fatal("trim should remove expired entries first")
	}
	if _, ok := cache.items.Load("oldest-live"); ok {
		t.Fatal("trim should evict the oldest live entry when over capacity")
	}
	if _, ok := cache.items.Load("newer-live"); !ok {
		t.Fatal("trim should keep newer live entries")
	}
	if _, ok := cache.items.Load("newest-live"); !ok {
		t.Fatal("trim should keep the newest live entry")
	}
}

func TestRingCacheReturnsSnapshotsInsteadOfInternalSlices(t *testing.T) {
	cache := NewMessageCache(2)
	cache.Add("alpha")
	cache.Add("beta")

	all := cache.GetAll()
	all[0] = "mutated"

	latest := cache.GetLast()
	if latest != "beta" {
		t.Fatalf("GetLast = %#v, want beta", latest)
	}
	if cache.GetAll()[0] != "alpha" {
		t.Fatal("GetAll should return a copy, not the internal slice")
	}

	tail := cache.GetN(1)
	tail[0] = "changed"
	if cache.GetLast() != "beta" {
		t.Fatal("GetN should return a copy, not a live view into the ring buffer")
	}
}
