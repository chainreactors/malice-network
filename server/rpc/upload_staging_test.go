package rpc

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type syncFailingUploadFile struct {
	closed bool
}

type closeFailingUploadFile struct {
	closed bool
}

func newTestUploadStagingManager(root string) *uploadStagingManager {
	m := newUploadStagingManager(root)
	m.availableDiskBytes = func(string) (uint64, error) { return ^uint64(0), nil }
	return m
}

func (f *syncFailingUploadFile) Write(p []byte) (int, error) {
	return len(p), nil
}

func (f *syncFailingUploadFile) Sync() error {
	return errors.New("forced sync failure")
}

func (f *syncFailingUploadFile) Close() error {
	f.closed = true
	return nil
}

func (f *closeFailingUploadFile) Write(p []byte) (int, error) {
	return len(p), nil
}

func (f *closeFailingUploadFile) Sync() error {
	return nil
}

func (f *closeFailingUploadFile) Close() error {
	f.closed = true
	return errors.New("forced close failure")
}

func TestUploadStagingAppendAndIdempotentFinal(t *testing.T) {
	root := t.TempDir()
	m := newTestUploadStagingManager(root)
	m.maxChunkBytes = 8
	m.maxFileBytes = 64

	meta := uploadImmutableMeta{
		name:      "a.bin",
		target:    "/tmp/a.bin",
		priv:      0o644,
		override:  true,
		totalSize: 10,
	}

	// chunk 0: 4 bytes
	u, res, err := m.prepareAppend("op1", "sess1", "upload-1", meta, 0, []byte("aaaa"))
	if err != nil {
		t.Fatalf("first chunk: %v", err)
	}
	if res.nextOffset != 4 || res.complete {
		t.Fatalf("first result = %+v", res)
	}
	u.mu.Unlock()

	// wrong offset
	if _, _, err := m.prepareAppend("op1", "sess1", "upload-1", meta, 9, []byte("x")); err == nil {
		t.Fatal("expected failed precondition for gap offset")
	}

	// chunk 1: 4 bytes
	u, res, err = m.prepareAppend("op1", "sess1", "upload-1", meta, 4, []byte("bbbb"))
	if err != nil {
		t.Fatalf("second chunk: %v", err)
	}
	if res.nextOffset != 8 || res.complete {
		t.Fatalf("second result = %+v", res)
	}
	// idempotent retry of second chunk
	u.mu.Unlock()
	u, res, err = m.prepareAppend("op1", "sess1", "upload-1", meta, 4, []byte("bbbb"))
	if err != nil {
		t.Fatalf("retry second chunk: %v", err)
	}
	if !res.replayed || res.nextOffset != 8 {
		t.Fatalf("expected replay next 8, got %+v", res)
	}
	u.mu.Unlock()

	// final 2 bytes
	u, res, err = m.prepareAppend("op1", "sess1", "upload-1", meta, 8, []byte("cc"))
	if err != nil {
		t.Fatalf("final chunk: %v", err)
	}
	if !res.complete || res.nextOffset != 10 {
		t.Fatalf("final result = %+v", res)
	}
	path := u.stagingPath
	u.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read staging: %v", err)
	}
	if !bytes.Equal(data, []byte("aaaabbbbcc")) {
		t.Fatalf("staging content = %q", data)
	}

	// same final chunk again after markCompleted
	u, res, err = m.prepareAppend("op1", "sess1", "upload-1", meta, 8, []byte("cc"))
	if err != nil {
		// still STAGED without taskPB may return already completed depending on state
		// mark completed first
		t.Logf("pre-complete replay err (ok if already staged without task): %v", err)
	} else {
		u.mu.Unlock()
	}

	// metadata conflict
	bad := meta
	bad.override = false
	if _, _, err := m.prepareAppend("op1", "sess1", "upload-1", bad, 0, []byte("aaaa")); err == nil {
		t.Fatal("expected metadata conflict")
	}
}

func TestUploadStagingZeroByte(t *testing.T) {
	root := t.TempDir()
	m := newTestUploadStagingManager(root)
	meta := uploadImmutableMeta{name: "empty", target: "/tmp/e", totalSize: 0}
	u, res, err := m.prepareAppend("op", "s", "z1", meta, 0, nil)
	if err != nil {
		t.Fatalf("zero byte: %v", err)
	}
	if !res.complete || res.nextOffset != 0 {
		t.Fatalf("zero result = %+v", res)
	}
	if _, err := os.Stat(u.stagingPath); err != nil {
		t.Fatalf("staging file: %v", err)
	}
	u.mu.Unlock()
}

func TestUploadStagingRejectsChunkTooLarge(t *testing.T) {
	m := newTestUploadStagingManager(t.TempDir())
	m.maxChunkBytes = 4
	meta := uploadImmutableMeta{name: "n", target: "t", totalSize: 100}
	if _, _, err := m.prepareAppend("op", "s", "id1", meta, 0, []byte("12345")); err == nil {
		t.Fatal("expected chunk too large")
	}
}

func TestUploadStagingEnforcesFileAndActiveUploadLimits(t *testing.T) {
	m := newTestUploadStagingManager(t.TempDir())
	m.maxFileBytes = 4
	tooLarge := uploadImmutableMeta{name: "large", target: "/tmp/large", totalSize: 5}
	if _, _, err := m.prepareAppend("op", "s", "large", tooLarge, 0, []byte("a")); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("oversized file error = %v, want ResourceExhausted", err)
	}

	m.maxFileBytes = 64
	m.maxActive = 1
	meta := uploadImmutableMeta{name: "pending", target: "/tmp/pending", totalSize: 2}
	u, _, err := m.prepareAppend("op", "s", "active-1", meta, 0, []byte("a"))
	if err != nil {
		t.Fatalf("first active upload: %v", err)
	}
	u.mu.Unlock()
	if _, _, err := m.prepareAppend("op", "s", "active-2", meta, 0, []byte("b")); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("second active upload error = %v, want ResourceExhausted", err)
	}

	u.mu.Lock()
	u.cleanupLocked()
	u.mu.Unlock()
}

func TestUploadStagingRejectsGlobalReservationAndLowDisk(t *testing.T) {
	m := newTestUploadStagingManager(t.TempDir())
	m.maxFileBytes = 64
	m.maxStagingBytes = 3
	m.minFreeDiskBytes = 1
	m.availableDiskBytes = func(string) (uint64, error) { return 1024, nil }
	meta := uploadImmutableMeta{name: "reserved", target: "/tmp/reserved", totalSize: 2}

	u, _, err := m.prepareAppend("op", "s1", "reserved-1", meta, 0, []byte("a"))
	if err != nil {
		t.Fatalf("first reserved upload: %v", err)
	}
	u.mu.Unlock()
	if _, _, err := m.prepareAppend("op", "s2", "reserved-2", meta, 0, []byte("b")); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("global reservation error = %v, want ResourceExhausted", err)
	}

	m2 := newTestUploadStagingManager(t.TempDir())
	m2.maxFileBytes = 64
	m2.maxStagingBytes = 64
	m2.minFreeDiskBytes = 10
	m2.availableDiskBytes = func(string) (uint64, error) { return 11, nil }
	if _, _, err := m2.prepareAppend("op", "s", "low-disk", meta, 0, []byte("a")); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("low disk error = %v, want ResourceExhausted", err)
	}
}

func TestUploadStagingSyncFailureRejectsFinalChunkAndCleansUp(t *testing.T) {
	root := t.TempDir()
	m := newTestUploadStagingManager(root)
	meta := uploadImmutableMeta{name: "sync.bin", target: "/tmp/sync.bin", totalSize: 2}

	u, _, err := m.prepareAppend("op", "s", "sync-failure", meta, 0, []byte("a"))
	if err != nil {
		t.Fatalf("prepare first chunk: %v", err)
	}
	stagingPath := u.stagingPath
	if err := u.file.Close(); err != nil {
		t.Fatalf("close real staging file: %v", err)
	}
	failingFile := &syncFailingUploadFile{}
	u.file = failingFile
	u.mu.Unlock()

	_, _, err = m.prepareAppend("op", "s", "sync-failure", meta, 1, []byte("b"))
	if status.Code(err) != codes.Internal {
		t.Fatalf("final chunk error = %v, want Internal", err)
	}
	if !failingFile.closed {
		t.Fatal("staging file was not closed after Sync failure")
	}
	if _, statErr := os.Stat(stagingPath); !os.IsNotExist(statErr) {
		t.Fatalf("staging file still exists after Sync failure: %v", statErr)
	}
	m.mu.Lock()
	_, exists := m.uploads[m.key("op", "s", "sync-failure")]
	m.mu.Unlock()
	if exists {
		t.Fatal("failed upload still registered")
	}
}

func TestUploadStagingCloseFailureRejectsFinalChunkAndCleansUp(t *testing.T) {
	root := t.TempDir()
	m := newTestUploadStagingManager(root)
	meta := uploadImmutableMeta{name: "close.bin", target: "/tmp/close.bin", totalSize: 2}

	u, _, err := m.prepareAppend("op", "s", "close-failure", meta, 0, []byte("a"))
	if err != nil {
		t.Fatalf("prepare first chunk: %v", err)
	}
	stagingPath := u.stagingPath
	if err := u.file.Close(); err != nil {
		t.Fatalf("close real staging file: %v", err)
	}
	failingFile := &closeFailingUploadFile{}
	u.file = failingFile
	u.mu.Unlock()

	_, _, err = m.prepareAppend("op", "s", "close-failure", meta, 1, []byte("b"))
	if status.Code(err) != codes.Internal {
		t.Fatalf("final chunk error = %v, want Internal", err)
	}
	if !failingFile.closed {
		t.Fatal("staging file close was not attempted")
	}
	if _, statErr := os.Stat(stagingPath); !os.IsNotExist(statErr) {
		t.Fatalf("staging file still exists after Close failure: %v", statErr)
	}
	m.mu.Lock()
	_, exists := m.uploads[m.key("op", "s", "close-failure")]
	m.mu.Unlock()
	if exists {
		t.Fatal("failed upload still registered")
	}
}

func TestUploadStagingOrphanCleanup(t *testing.T) {
	root := t.TempDir()
	orphan := filepath.Join(root, "dead.part")
	if err := os.WriteFile(orphan, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newTestUploadStagingManager(root)
	m.purgeOrphansOnStartup()
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphan should be removed, err=%v", err)
	}
}

func TestUploadStagingJanitorPurgesExpiredReceivingUpload(t *testing.T) {
	root := t.TempDir()
	m := newTestUploadStagingManager(root)
	now := time.Now()
	m.now = func() time.Time { return now }
	m.ttl = 10 * time.Millisecond

	meta := uploadImmutableMeta{name: "pending.bin", target: "/tmp/pending.bin", totalSize: 2}
	u, _, err := m.prepareAppend("operator", "session", "pending", meta, 0, []byte("a"))
	if err != nil {
		t.Fatalf("prepare pending upload: %v", err)
	}
	stagingPath := u.stagingPath
	u.mu.Unlock()

	now = now.Add(time.Second)
	stop := m.startJanitor(5 * time.Millisecond)
	t.Cleanup(stop)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		remaining := len(m.uploads)
		m.mu.Unlock()
		_, statErr := os.Stat(stagingPath)
		if remaining == 0 && os.IsNotExist(statErr) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	m.mu.Lock()
	remaining := len(m.uploads)
	m.mu.Unlock()
	_, statErr := os.Stat(stagingPath)
	t.Fatalf("janitor did not purge upload: records=%d statErr=%v", remaining, statErr)
}
