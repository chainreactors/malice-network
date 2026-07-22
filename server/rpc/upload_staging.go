package rpc

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/chainreactors/malice-network/server/internal/configs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// Runtime cleanup cadence; substantially shorter than the staging TTL.
	uploadStagingJanitorInterval = time.Minute
)

type uploadStagingState string

const (
	uploadStateReceiving  uploadStagingState = "RECEIVING"
	uploadStateStaged     uploadStagingState = "STAGED"
	uploadStateDelivering uploadStagingState = "DELIVERING"
	uploadStateCompleted  uploadStagingState = "COMPLETED"
	uploadStateFailed     uploadStagingState = "FAILED"
	uploadStateExpired    uploadStagingState = "EXPIRED"
)

type uploadImmutableMeta struct {
	name      string
	target    string
	priv      uint32
	hidden    bool
	override  bool
	totalSize uint64
}

type lastChunkRecord struct {
	offset uint64
	length int
	digest string
}

type uploadStagingFile interface {
	Write([]byte) (int, error)
	Sync() error
	Close() error
}

type stagedUpload struct {
	mu sync.Mutex

	uploadID   string
	internalID string
	identity   string
	sessionID  string
	meta       uploadImmutableMeta

	nextOffset  uint64
	stagingPath string
	file        uploadStagingFile
	lastChunk   *lastChunkRecord
	state       uploadStagingState

	// Set once when the final chunk creates the implant Task.
	taskPB any // *clientpb.Task stored as any to avoid import cycle in helpers; cast at use site

	createdAt time.Time
	updatedAt time.Time
	expiresAt time.Time
}

type uploadStagingKey struct {
	identity  string
	sessionID string
	uploadID  string
}

type uploadStagingManager struct {
	mu       sync.Mutex
	uploads  map[uploadStagingKey]*stagedUpload
	tempRoot string
	now      func() time.Time
	// maxChunkBytes / maxFileBytes are overridable in tests.
	maxChunkBytes      uint64
	maxFileBytes       uint64
	maxStagingBytes    uint64
	minFreeDiskBytes   uint64
	ttl                time.Duration
	maxActive          int
	availableDiskBytes func(string) (uint64, error)
	janitorOnce        sync.Once
	janitorStopOnce    sync.Once
	janitorStop        chan struct{}
	janitorDone        chan struct{}
}

func newUploadStagingManager(tempRoot string) *uploadStagingManager {
	return &uploadStagingManager{
		uploads:            make(map[uploadStagingKey]*stagedUpload),
		tempRoot:           tempRoot,
		now:                time.Now,
		maxChunkBytes:      configs.DefaultUploadMaxChunkBytes,
		maxFileBytes:       configs.DefaultUploadMaxFileBytes,
		maxStagingBytes:    configs.DefaultUploadMaxStagingBytes,
		minFreeDiskBytes:   configs.DefaultUploadMinFreeDiskBytes,
		ttl:                time.Duration(configs.DefaultUploadStagingTTLSeconds) * time.Second,
		maxActive:          configs.DefaultUploadMaxActivePerSession,
		availableDiskBytes: availableDiskBytes,
	}
}

var globalUploadStaging = newUploadStagingManager("")

func uploadStagingRoot() string {
	if globalUploadStaging.tempRoot != "" {
		return globalUploadStaging.tempRoot
	}
	return filepath.Join(configs.TempPath, "uploads")
}

func (m *uploadStagingManager) key(identity, sessionID, uploadID string) uploadStagingKey {
	return uploadStagingKey{identity: identity, sessionID: sessionID, uploadID: uploadID}
}

func (m *uploadStagingManager) countActiveLocked(identity, sessionID string) int {
	n := 0
	for k, u := range m.uploads {
		if k.identity != identity || k.sessionID != sessionID {
			continue
		}
		u.mu.Lock()
		st := u.state
		u.mu.Unlock()
		if st == uploadStateReceiving {
			n++
		}
	}
	return n
}

func saturatingAddUint64(a, b uint64) uint64 {
	if ^uint64(0)-a < b {
		return ^uint64(0)
	}
	return a + b
}

func exceedsUint64Limit(limit, current, additional uint64) bool {
	return limit > 0 && (current > limit || additional > limit-current)
}

func (m *uploadStagingManager) stagingReservationsLocked() (reserved, remaining uint64) {
	for _, u := range m.uploads {
		u.mu.Lock()
		if u.state == uploadStateReceiving || u.state == uploadStateStaged || u.state == uploadStateDelivering {
			reserved = saturatingAddUint64(reserved, u.meta.totalSize)
			if u.nextOffset < u.meta.totalSize {
				remaining = saturatingAddUint64(remaining, u.meta.totalSize-u.nextOffset)
			}
		}
		u.mu.Unlock()
	}
	return reserved, remaining
}

func (m *uploadStagingManager) applyConfig(cfg *configs.UploadConfig) {
	if cfg == nil {
		return
	}
	m.mu.Lock()
	m.maxChunkBytes = cfg.MaxChunkBytes
	m.maxFileBytes = cfg.MaxFileBytes
	m.maxStagingBytes = cfg.MaxStagingBytes
	m.minFreeDiskBytes = cfg.MinFreeDiskBytes
	m.maxActive = cfg.MaxActivePerSession
	m.ttl = time.Duration(cfg.StagingTTLSeconds) * time.Second
	m.mu.Unlock()
}

func validateUploadID(uploadID string) error {
	if uploadID == "" {
		return status.Error(codes.InvalidArgument, "upload_id is required")
	}
	if len(uploadID) > 128 {
		return status.Error(codes.InvalidArgument, "upload_id too long")
	}
	for _, r := range uploadID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return status.Error(codes.InvalidArgument, "upload_id has invalid characters")
	}
	return nil
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func randomInternalID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// appendChunkResult is returned to the RPC handler after a successful append.
type appendChunkResult struct {
	nextOffset uint64
	// complete is true when nextOffset == totalSize after this chunk.
	complete bool
	// upload is locked by the caller via WithUpload / Complete helpers.
	upload *stagedUpload
	// replayed is true when this request was an idempotent retry of the last chunk.
	replayed bool
}

// prepareAppend validates and either creates or loads the staging session for
// the incoming chunk. The returned stagedUpload is locked; caller must Unlock.
func (m *uploadStagingManager) prepareAppend(
	identity, sessionID, uploadID string,
	meta uploadImmutableMeta,
	offset uint64,
	data []byte,
) (*stagedUpload, *appendChunkResult, error) {
	if err := validateUploadID(uploadID); err != nil {
		return nil, nil, err
	}
	if identity == "" {
		return nil, nil, status.Error(codes.Unauthenticated, "missing operator identity")
	}
	if sessionID == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	if meta.name == "" || meta.target == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "name and target are required")
	}
	if uint64(len(data)) > m.maxChunkBytes {
		return nil, nil, status.Errorf(codes.InvalidArgument, "chunk too large: %d > %d", len(data), m.maxChunkBytes)
	}
	if meta.totalSize > m.maxFileBytes {
		return nil, nil, status.Errorf(codes.ResourceExhausted, "file too large: %d > %d", meta.totalSize, m.maxFileBytes)
	}
	if offset > meta.totalSize {
		return nil, nil, status.Errorf(codes.InvalidArgument, "offset %d beyond total_size %d", offset, meta.totalSize)
	}
	if uint64(len(data)) > meta.totalSize-offset {
		return nil, nil, status.Errorf(codes.InvalidArgument, "chunk overruns total_size")
	}
	// Non-zero files must not send empty intermediate chunks.
	if meta.totalSize > 0 && len(data) == 0 {
		return nil, nil, status.Error(codes.InvalidArgument, "empty chunk not allowed for non-zero total_size")
	}
	// Zero-byte upload: only empty data at offset 0.
	if meta.totalSize == 0 {
		if offset != 0 || len(data) != 0 {
			return nil, nil, status.Error(codes.InvalidArgument, "zero-byte upload must use offset=0 and empty data")
		}
	}

	now := m.now()
	key := m.key(identity, sessionID, uploadID)

	m.mu.Lock()
	u, exists := m.uploads[key]
	if !exists {
		if offset != 0 {
			m.mu.Unlock()
			return nil, nil, status.Errorf(codes.FailedPrecondition, "unknown upload_id or expected offset 0, got %d", offset)
		}
		if m.countActiveLocked(identity, sessionID) >= m.maxActive {
			m.mu.Unlock()
			return nil, nil, status.Error(codes.ResourceExhausted, "too many active uploads for session")
		}
		internalID, err := randomInternalID()
		if err != nil {
			m.mu.Unlock()
			return nil, nil, status.Errorf(codes.Internal, "allocate upload id: %v", err)
		}
		root := m.tempRoot
		if root == "" {
			root = uploadStagingRoot()
		}
		if err := os.MkdirAll(root, 0o700); err != nil {
			m.mu.Unlock()
			return nil, nil, status.Errorf(codes.Internal, "create staging dir: %v", err)
		}
		reserved, remaining := m.stagingReservationsLocked()
		if exceedsUint64Limit(m.maxStagingBytes, reserved, meta.totalSize) {
			m.mu.Unlock()
			return nil, nil, status.Error(codes.ResourceExhausted, "staging quota exceeded")
		}
		available, err := m.availableDiskBytes(root)
		if err != nil {
			m.mu.Unlock()
			return nil, nil, status.Errorf(codes.Internal, "inspect staging disk: %v", err)
		}
		required := saturatingAddUint64(m.minFreeDiskBytes, remaining)
		required = saturatingAddUint64(required, meta.totalSize)
		if available < required {
			m.mu.Unlock()
			return nil, nil, status.Error(codes.ResourceExhausted, "insufficient staging disk space")
		}
		stagingPath := filepath.Join(root, internalID+".part")
		f, err := os.OpenFile(stagingPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if err != nil {
			m.mu.Unlock()
			return nil, nil, status.Errorf(codes.Internal, "create staging file: %v", err)
		}
		u = &stagedUpload{
			uploadID:    uploadID,
			internalID:  internalID,
			identity:    identity,
			sessionID:   sessionID,
			meta:        meta,
			nextOffset:  0,
			stagingPath: stagingPath,
			file:        f,
			state:       uploadStateReceiving,
			createdAt:   now,
			updatedAt:   now,
			expiresAt:   now.Add(m.ttl),
		}
		m.uploads[key] = u
	}
	m.mu.Unlock()

	u.mu.Lock()
	// Expired?
	if now.After(u.expiresAt) && u.state == uploadStateReceiving {
		u.state = uploadStateExpired
		u.cleanupLocked()
		u.mu.Unlock()
		m.remove(key)
		return nil, nil, status.Error(codes.NotFound, "upload expired")
	}

	// Completed/failed with task: only allow final-chunk idempotent replay.
	if u.state == uploadStateCompleted || u.state == uploadStateStaged || u.state == uploadStateDelivering || u.state == uploadStateFailed {
		if u.taskPB != nil && u.lastChunk != nil &&
			offset == u.lastChunk.offset &&
			len(data) == u.lastChunk.length &&
			digestBytes(data) == u.lastChunk.digest &&
			metaMatches(u.meta, meta) {
			res := &appendChunkResult{
				nextOffset: u.nextOffset,
				complete:   true,
				upload:     u,
				replayed:   true,
			}
			// keep locked for caller to read taskPB
			return u, res, nil
		}
		state := u.state
		u.mu.Unlock()
		if state == uploadStateFailed {
			return nil, nil, status.Error(codes.FailedPrecondition, "upload previously failed")
		}
		return nil, nil, status.Error(codes.AlreadyExists, "upload already completed")
	}

	if u.state != uploadStateReceiving {
		u.mu.Unlock()
		return nil, nil, status.Errorf(codes.FailedPrecondition, "upload state %s does not accept chunks", u.state)
	}

	if !metaMatches(u.meta, meta) {
		u.mu.Unlock()
		return nil, nil, status.Error(codes.AlreadyExists, "upload_id metadata conflict")
	}

	// Idempotent retry of the most recent accepted chunk.
	if u.lastChunk != nil &&
		offset == u.lastChunk.offset &&
		len(data) == u.lastChunk.length &&
		digestBytes(data) == u.lastChunk.digest {
		res := &appendChunkResult{
			nextOffset: u.nextOffset,
			complete:   u.nextOffset == u.meta.totalSize,
			upload:     u,
			replayed:   true,
		}
		return u, res, nil
	}

	if offset != u.nextOffset {
		expected := u.nextOffset
		u.mu.Unlock()
		return nil, nil, status.Errorf(codes.FailedPrecondition, "expected offset %d, got %d", expected, offset)
	}

	// Append bytes (zero-byte complete has empty data).
	if len(data) > 0 {
		available, diskErr := m.availableDiskBytes(filepath.Dir(u.stagingPath))
		required := saturatingAddUint64(m.minFreeDiskBytes, uint64(len(data)))
		if diskErr != nil || available < required {
			u.state = uploadStateFailed
			u.cleanupLocked()
			u.mu.Unlock()
			m.remove(key)
			if diskErr != nil {
				return nil, nil, status.Errorf(codes.Internal, "inspect staging disk: %v", diskErr)
			}
			return nil, nil, status.Error(codes.ResourceExhausted, "insufficient staging disk space")
		}
		if _, err := u.file.Write(data); err != nil {
			u.state = uploadStateFailed
			u.cleanupLocked()
			u.mu.Unlock()
			m.remove(key)
			return nil, nil, status.Errorf(codes.Internal, "write staging: %v", err)
		}
	}

	u.lastChunk = &lastChunkRecord{
		offset: offset,
		length: len(data),
		digest: digestBytes(data),
	}
	u.nextOffset = offset + uint64(len(data))
	u.updatedAt = now
	u.expiresAt = now.Add(m.ttl)

	complete := u.nextOffset == u.meta.totalSize
	if complete {
		// Flush and close before Task creation; keep path for reader dispatch.
		if u.file != nil {
			if err := u.file.Sync(); err != nil {
				u.state = uploadStateFailed
				u.cleanupLocked()
				u.mu.Unlock()
				m.remove(key)
				return nil, nil, status.Errorf(codes.Internal, "sync staging: %v", err)
			}
			if err := u.file.Close(); err != nil {
				u.state = uploadStateFailed
				u.cleanupLocked()
				u.mu.Unlock()
				m.remove(key)
				return nil, nil, status.Errorf(codes.Internal, "close staging: %v", err)
			}
			u.file = nil
		}
		u.state = uploadStateStaged
	}

	res := &appendChunkResult{
		nextOffset: u.nextOffset,
		complete:   complete,
		upload:     u,
		replayed:   false,
	}
	return u, res, nil
}

func metaMatches(a, b uploadImmutableMeta) bool {
	return a.name == b.name &&
		a.target == b.target &&
		a.priv == b.priv &&
		a.hidden == b.hidden &&
		a.override == b.override &&
		a.totalSize == b.totalSize
}

func (u *stagedUpload) cleanupLocked() {
	if u.file != nil {
		_ = u.file.Close()
		u.file = nil
	}
	if u.stagingPath != "" {
		_ = os.Remove(u.stagingPath)
	}
}

func (m *uploadStagingManager) remove(key uploadStagingKey) {
	m.mu.Lock()
	delete(m.uploads, key)
	m.mu.Unlock()
}

// markDelivering transitions STAGED -> DELIVERING. Caller must hold u.mu.
func (u *stagedUpload) markDelivering() error {
	if u.state != uploadStateStaged && u.state != uploadStateDelivering {
		return fmt.Errorf("cannot deliver from state %s", u.state)
	}
	u.state = uploadStateDelivering
	return nil
}

// markDispatched stores the Task for final-chunk retries while downstream
// delivery is still running. Caller must hold u.mu.
func (u *stagedUpload) markDispatched(task any, now time.Time) {
	u.taskPB = task
	u.state = uploadStateDelivering
	u.updatedAt = now
}

// markCompleted records downstream task termination and starts a fresh TTL for
// idempotent final-chunk replays. Caller must hold u.mu.
func (u *stagedUpload) markCompleted(now time.Time, ttl time.Duration) {
	u.state = uploadStateCompleted
	u.updatedAt = now
	u.expiresAt = now.Add(ttl)
	u.cleanupLocked()
}

func (u *stagedUpload) markFailed() {
	u.state = uploadStateFailed
	u.cleanupLocked()
}

// purgeExpired removes RECEIVING uploads past TTL. Safe to call periodically.
func (m *uploadStagingManager) purgeExpired() {
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, u := range m.uploads {
		u.mu.Lock()
		expired := now.After(u.expiresAt)
		if expired && (u.state == uploadStateReceiving || u.state == uploadStateFailed ||
			(u.state == uploadStateCompleted && now.After(u.expiresAt))) {
			if u.state == uploadStateReceiving {
				u.state = uploadStateExpired
			}
			u.cleanupLocked()
			delete(m.uploads, k)
		}
		u.mu.Unlock()
	}
}

// purgeOrphansOnStartup deletes .part files under the uploads root that are not
// tracked in memory (process restart). First version does not resume.
func (m *uploadStagingManager) purgeOrphansOnStartup() {
	root := m.tempRoot
	if root == "" {
		root = uploadStagingRoot()
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	m.mu.Lock()
	known := make(map[string]struct{}, len(m.uploads))
	for _, u := range m.uploads {
		known[u.stagingPath] = struct{}{}
	}
	m.mu.Unlock()
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(root, e.Name())
		if _, ok := known[path]; ok {
			continue
		}
		if filepath.Ext(e.Name()) == ".part" {
			_ = os.Remove(path)
		}
	}
}

// startJanitor wires orphan cleanup and recurring TTL cleanup into runtime.
// It is safe to call more than once; the returned function stops the janitor.
func (m *uploadStagingManager) startJanitor(interval time.Duration) func() {
	if interval <= 0 {
		interval = uploadStagingJanitorInterval
	}
	m.janitorOnce.Do(func() {
		m.janitorStop = make(chan struct{})
		m.janitorDone = make(chan struct{})
		m.purgeOrphansOnStartup()
		go func() {
			defer close(m.janitorDone)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					m.purgeExpired()
				case <-m.janitorStop:
					return
				}
			}
		}()
	})

	return func() {
		m.janitorStopOnce.Do(func() {
			if m.janitorStop != nil {
				close(m.janitorStop)
			}
		})
		if m.janitorDone != nil {
			<-m.janitorDone
		}
	}
}
