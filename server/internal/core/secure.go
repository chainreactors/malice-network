package core

import (
	"sync"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

// SecureManager handles key exchange request and response processing.
type SecureManager struct {
	sessionID string

	mu            sync.Mutex
	keyPair       *clientpb.KeyPair
	keyCounter    uint32
	rotationCount uint32
}

// NewSecureSpiteManager creates a new secure manager.
func NewSecureSpiteManager(sess *Session) *SecureManager {
	return &SecureManager{
		keyPair:       sess.KeyPair,
		sessionID:     sess.ID,
		keyCounter:    0,
		rotationCount: 100,
	}
}

// UpdateKeyPair updates the active key pair reference.
func (s *SecureManager) UpdateKeyPair(keyPair *clientpb.KeyPair) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyPair = keyPair
}

// ShouldRotateKey checks whether key rotation is needed.
func (s *SecureManager) ShouldRotateKey() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.keyCounter >= s.rotationCount
}

// IncrementCounter increments the message counter.
func (s *SecureManager) IncrementCounter() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyCounter++
}

// ResetCounters resets the counter (called after key exchange completes).
func (s *SecureManager) ResetCounters() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyCounter = 0
}
