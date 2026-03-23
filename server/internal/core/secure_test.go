package core

import (
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func newSecureTestSession() *Session {
	return &Session{
		ID:   "secure-test-session",
		Type: "test",
		SessionContext: &client.SessionContext{
			SessionInfo: &client.SessionInfo{},
			KeyPair: &clientpb.KeyPair{
				PublicKey:  "test-pub-key",
				PrivateKey: "test-priv-key",
			},
		},
	}
}

// TestNewSecureSpiteManager_Defaults verifies initial state after construction.
func TestNewSecureSpiteManager_Defaults(t *testing.T) {
	t.Parallel()
	sess := newSecureTestSession()
	sm := NewSecureSpiteManager(sess)

	if sm.keyCounter != 0 {
		t.Errorf("keyCounter = %d, want 0", sm.keyCounter)
	}
	if sm.rotationCount != 100 {
		t.Errorf("rotationCount = %d, want 100", sm.rotationCount)
	}
	if sm.sessionID != sess.ID {
		t.Errorf("sessionID = %q, want %q", sm.sessionID, sess.ID)
	}
	if sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be false at counter=0")
	}
}

// TestSecureManager_IncrementAndCheck verifies the rotation threshold at exactly 100.
func TestSecureManager_IncrementAndCheck(t *testing.T) {
	t.Parallel()
	sess := newSecureTestSession()
	sm := NewSecureSpiteManager(sess)

	// Increment 99 times - should NOT trigger rotation
	for i := 0; i < 99; i++ {
		sm.IncrementCounter()
	}
	if sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be false at counter=99")
	}

	// One more increment to reach 100 - should trigger
	sm.IncrementCounter()
	if !sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be true at counter=100")
	}
}

// TestSecureManager_IncrementBeyondThreshold verifies ShouldRotateKey stays true
// after exceeding the threshold (counter keeps growing without reset).
func TestSecureManager_IncrementBeyondThreshold(t *testing.T) {
	t.Parallel()
	sess := newSecureTestSession()
	sm := NewSecureSpiteManager(sess)

	for i := 0; i < 150; i++ {
		sm.IncrementCounter()
	}
	if !sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be true at counter=150")
	}
}

// TestSecureManager_ResetCounters verifies counter resets to 0.
func TestSecureManager_ResetCounters(t *testing.T) {
	t.Parallel()
	sess := newSecureTestSession()
	sm := NewSecureSpiteManager(sess)

	for i := 0; i < 50; i++ {
		sm.IncrementCounter()
	}
	sm.ResetCounters()

	if sm.keyCounter != 0 {
		t.Errorf("keyCounter after reset = %d, want 0", sm.keyCounter)
	}
	if sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be false after reset")
	}
}

// TestSecureManager_ResetAfterRotation verifies the full rotation cycle:
// increment to threshold, detect rotation needed, reset, verify no longer needed.
func TestSecureManager_RotationBoundary(t *testing.T) {
	t.Parallel()
	sess := newSecureTestSession()
	sm := NewSecureSpiteManager(sess)

	// Increment to exactly rotationCount
	for i := 0; i < 100; i++ {
		sm.IncrementCounter()
	}
	if !sm.ShouldRotateKey() {
		t.Fatal("ShouldRotateKey should be true at threshold")
	}

	// Reset
	sm.ResetCounters()
	if sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be false after reset")
	}

	// Increment again to threshold
	for i := 0; i < 100; i++ {
		sm.IncrementCounter()
	}
	if !sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be true again after second cycle")
	}
}

// TestSecureManager_UpdateKeyPair verifies key pair update does not deadlock
// and the manager continues to function afterward.
func TestSecureManager_UpdateKeyPair(t *testing.T) {
	t.Parallel()
	sess := newSecureTestSession()
	sm := NewSecureSpiteManager(sess)

	newKP := &clientpb.KeyPair{
		PublicKey:  "new-pub-key",
		PrivateKey: "new-priv-key",
	}

	// Update should not deadlock
	sm.UpdateKeyPair(newKP)

	// Verify the manager still works after update
	sm.IncrementCounter()
	if sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be false after 1 increment")
	}

	// Verify key pair was updated
	sm.mu.Lock()
	if sm.keyPair.PublicKey != "new-pub-key" {
		t.Errorf("PublicKey = %q, want %q", sm.keyPair.PublicKey, "new-pub-key")
	}
	if sm.keyPair.PrivateKey != "new-priv-key" {
		t.Errorf("PrivateKey = %q, want %q", sm.keyPair.PrivateKey, "new-priv-key")
	}
	sm.mu.Unlock()
}

// TestSecureManager_UpdateKeyPairNil verifies updating with nil key pair does not panic.
func TestSecureManager_UpdateKeyPairNil(t *testing.T) {
	t.Parallel()
	sess := newSecureTestSession()
	sm := NewSecureSpiteManager(sess)

	// Should not panic
	sm.UpdateKeyPair(nil)

	sm.mu.Lock()
	if sm.keyPair != nil {
		t.Error("keyPair should be nil after nil update")
	}
	sm.mu.Unlock()

	// Manager should still function
	sm.IncrementCounter()
	if sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should still work after nil key pair update")
	}
}

// TestSecureManager_ConcurrentIncrementAndRotate launches multiple goroutines
// that concurrently increment, check rotation, and reset. Run with -race to
// detect data races.
func TestSecureManager_ConcurrentIncrementAndRotate(t *testing.T) {
	t.Parallel()
	sess := newSecureTestSession()
	sm := NewSecureSpiteManager(sess)

	var wg sync.WaitGroup

	// 100 goroutines incrementing
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm.IncrementCounter()
		}()
	}

	// 10 goroutines checking rotation
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = sm.ShouldRotateKey()
		}()
	}

	// 5 goroutines resetting
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm.ResetCounters()
		}()
	}

	// 5 goroutines updating key pair
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm.UpdateKeyPair(&clientpb.KeyPair{
				PublicKey:  "concurrent-pub",
				PrivateKey: "concurrent-priv",
			})
		}()
	}

	wg.Wait()

	// After all goroutines complete, the manager should still be functional
	sm.ResetCounters()
	if sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be false after final reset")
	}
}

// TestSecureManager_ResetCountersIdempotent verifies resetting multiple times
// in a row does not cause issues.
func TestSecureManager_ResetCountersIdempotent(t *testing.T) {
	t.Parallel()
	sess := newSecureTestSession()
	sm := NewSecureSpiteManager(sess)

	sm.IncrementCounter()
	sm.ResetCounters()
	sm.ResetCounters()
	sm.ResetCounters()

	if sm.keyCounter != 0 {
		t.Errorf("keyCounter = %d after multiple resets, want 0", sm.keyCounter)
	}
}

// TestSecureManager_CounterOverflow verifies behavior if counter is incremented
// a very large number of times without reset (uint32 overflow scenario).
// This is a theoretical edge case since rotation should happen at 100.
func TestSecureManager_CounterOverflow(t *testing.T) {
	t.Parallel()
	sess := newSecureTestSession()
	sm := NewSecureSpiteManager(sess)

	// Directly set counter near uint32 max to test overflow
	sm.mu.Lock()
	sm.keyCounter = ^uint32(0) // 4294967295
	sm.mu.Unlock()

	// Should definitely need rotation
	if !sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be true at max uint32")
	}

	// One more increment wraps to 0
	sm.IncrementCounter()
	sm.mu.Lock()
	counter := sm.keyCounter
	sm.mu.Unlock()

	if counter != 0 {
		t.Errorf("expected counter overflow to 0, got %d", counter)
	}

	// After overflow, ShouldRotateKey should be false (counter=0 < rotationCount=100)
	if sm.ShouldRotateKey() {
		t.Error("ShouldRotateKey should be false after counter overflow to 0")
	}
}

// TestSecureManager_NilKeyPairSession verifies creating a SecureManager with
// a session that has a nil KeyPair does not panic.
func TestSecureManager_NilKeyPairSession(t *testing.T) {
	t.Parallel()
	sess := &Session{
		ID:   "nil-keypair-session",
		Type: "test",
		SessionContext: &client.SessionContext{
			SessionInfo: &client.SessionInfo{},
			KeyPair:     nil,
		},
	}

	sm := NewSecureSpiteManager(sess)
	if sm == nil {
		t.Fatal("NewSecureSpiteManager returned nil")
	}

	sm.mu.Lock()
	if sm.keyPair != nil {
		t.Error("keyPair should be nil for session with nil KeyPair")
	}
	sm.mu.Unlock()

	// Should still function for counter operations
	sm.IncrementCounter()
	sm.ResetCounters()
}
