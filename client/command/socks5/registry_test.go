package socks5

import (
	"net"
	"testing"
	"time"
)

func TestRegistryMultiListenerPerSession(t *testing.T) {
	// Isolate global registry for this test.
	old := registry
	registry = newSocksRegistry()
	t.Cleanup(func() { registry = old })

	sid := "session-aaaaaaaa"
	mk := func(bind string, port int) *SocksService {
		return &SocksService{
			id:        shortID(sid, bind, port),
			sessionID: sid,
			bind:      bind,
			port:      port,
			user:      "u",
			status:    StatusListening,
			createdAt: time.Now(),
			conns:     make(map[uint32]*localConn),
			stopCh:    make(chan struct{}),
		}
	}

	s1 := mk("127.0.0.1", 18001)
	s2 := mk("127.0.0.1", 18002)
	if err := registry.addService(s1); err != nil {
		t.Fatalf("add s1: %v", err)
	}
	if err := registry.addService(s2); err != nil {
		t.Fatalf("add s2: %v", err)
	}
	// duplicate port same session
	if err := registry.addService(mk("127.0.0.1", 18001)); err == nil {
		t.Fatal("expected duplicate port error")
	}
	// 0.0.0.0 conflicts with same port
	if err := registry.addService(mk("0.0.0.0", 18001)); err == nil {
		t.Fatal("expected wildcard port conflict")
	}

	list := registry.list(sid)
	if len(list) != 2 {
		t.Fatalf("list session want 2 got %d", len(list))
	}
	if len(registry.list("")) != 2 {
		t.Fatalf("global list want 2")
	}

	// second session
	s3 := &SocksService{
		id: shortID("session-bbbbbbbb", "127.0.0.1", 18003), sessionID: "session-bbbbbbbb",
		bind: "127.0.0.1", port: 18003, status: StatusListening, conns: map[uint32]*localConn{}, stopCh: make(chan struct{}),
	}
	if err := registry.addService(s3); err != nil {
		t.Fatal(err)
	}
	if len(registry.list("")) != 3 {
		t.Fatalf("global want 3 got %d", len(registry.list("")))
	}
	if len(registry.list(sid)) != 2 {
		t.Fatalf("filter session want 2")
	}

	empty := registry.removeService(s1)
	if empty {
		t.Fatal("session should still have s2")
	}
	empty = registry.removeService(s2)
	if !empty {
		t.Fatal("session should be empty after removing both")
	}
	if registry.getByPort(sid, 18002) != nil {
		t.Fatal("removed service still visible")
	}
}

func TestRegistrySharedRelayNextID(t *testing.T) {
	old := registry
	registry = newSocksRegistry()
	t.Cleanup(func() { registry = old })

	rel := registry.getRelay("sess-1")
	a := rel.nextID.Add(1) - 1
	b := rel.nextID.Add(1) - 1
	if a == b {
		t.Fatalf("conn ids should differ %d %d", a, b)
	}
	// same session returns same relay
	if registry.getRelay("sess-1") != rel {
		t.Fatal("relay not shared")
	}
}

func TestServiceKeyAndListSort(t *testing.T) {
	old := registry
	registry = newSocksRegistry()
	t.Cleanup(func() { registry = old })

	for _, p := range []int{1082, 1080, 1081} {
		svc := &SocksService{
			id: shortID("abc", "127.0.0.1", p), sessionID: "abc",
			bind: "127.0.0.1", port: p, status: StatusListening,
			conns: map[uint32]*localConn{}, stopCh: make(chan struct{}),
		}
		if err := registry.addService(svc); err != nil {
			t.Fatal(err)
		}
	}
	list := registry.list("abc")
	if list[0].port != 1080 || list[1].port != 1081 || list[2].port != 1082 {
		t.Fatalf("ports not sorted: %d %d %d", list[0].port, list[1].port, list[2].port)
	}
}

// Ensure we can bind two real listeners (sanity for multi-port).
func TestDualLocalListen(t *testing.T) {
	ln1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln1.Close()
	ln2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln2.Close()
	if ln1.Addr().String() == ln2.Addr().String() {
		t.Fatal("ports should differ")
	}
}
