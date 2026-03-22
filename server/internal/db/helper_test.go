package db

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

// initTestDB initializes a fresh SQLite test database.
// It reuses the setupTestDB helper from sql_test.go to prepare paths,
// then calls NewDBClient to create & migrate the schema.
func initTestDB(t *testing.T) {
	t.Helper()
	setupTestDB(t)
	var err error
	Client, err = NewDBClient(nil)
	if err != nil {
		t.Fatalf("failed to initialize test DB client: %v", err)
	}
}

// ============================================
// Collection ToProtobuf Tests
// ============================================

func TestSessions_ToProtobuf_Empty(t *testing.T) {
	var s Sessions
	pb := s.ToProtobuf()
	if pb == nil {
		t.Fatal("ToProtobuf should not return nil for empty slice")
	}
	if len(pb.Sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(pb.Sessions))
	}
}

func TestTasks_ToProtobuf_NilElement(t *testing.T) {
	tasks := Tasks{nil}
	pb := tasks.ToProtobuf()
	if len(pb.Tasks) != 0 {
		t.Errorf("nil elements should be skipped, got %d", len(pb.Tasks))
	}
}

func TestPipelines_ToProtobuf(t *testing.T) {
	pipelines := Pipelines{
		&models.Pipeline{
			Name:       "test-tcp",
			ListenerId: "listener1",
			Type:       consts.TCPPipeline,
			PipelineParams: &implanttypes.PipelineParams{
				Tls:        &implanttypes.TlsConfig{},
				Encryption: implanttypes.EncryptionsConfig{},
			},
		},
	}
	pb := pipelines.ToProtobuf()
	if len(pb.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pb.Pipelines))
	}
	if pb.Pipelines[0].Name != "test-tcp" {
		t.Errorf("expected name 'test-tcp', got %q", pb.Pipelines[0].Name)
	}
}

func TestOperators_ToProtobuf(t *testing.T) {
	operators := Operators{
		&models.Operator{Name: "alice", Type: "client"},
	}
	pb := operators.ToProtobuf()
	if len(pb.Clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(pb.Clients))
	}
	if pb.Clients[0].Name != "alice" {
		t.Errorf("expected name 'alice', got %q", pb.Clients[0].Name)
	}
}

func TestProfiles_ToProtobuf(t *testing.T) {
	profiles := Profiles{
		&models.Profile{Name: "default"},
	}
	pb := profiles.ToProtobuf()
	if len(pb.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(pb.Profiles))
	}
	if pb.Profiles[0].Name != "default" {
		t.Errorf("expected name 'default', got %q", pb.Profiles[0].Name)
	}
}

func TestArtifacts_ToProtobuf(t *testing.T) {
	artifacts := Artifacts{
		&models.Artifact{Name: "build1"},
	}
	pb := artifacts.ToProtobuf()
	if len(pb.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(pb.Artifacts))
	}
	if pb.Artifacts[0].Name != "build1" {
		t.Errorf("expected name 'build1', got %q", pb.Artifacts[0].Name)
	}
}

// ============================================
// Generic CRUD Tests
// ============================================

func TestSave_And_Delete(t *testing.T) {
	initTestDB(t)

	op := &models.Operator{Name: "test-crud-op", Type: "client"}
	if err := Save(op); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify it exists
	found, err := FindOperatorByName("test-crud-op")
	if err != nil {
		t.Fatalf("FindOperatorByName failed: %v", err)
	}
	if found.Name != "test-crud-op" {
		t.Errorf("expected name 'test-crud-op', got %q", found.Name)
	}

	// Update via Save
	found.Type = "listener"
	if err := Save(found); err != nil {
		t.Fatalf("Save (update) failed: %v", err)
	}

	// Verify update
	updated, _ := FindOperatorByName("test-crud-op")
	if updated.Type != "listener" {
		t.Errorf("expected type 'listener', got %q", updated.Type)
	}
}

// ============================================
// SessionQuery Builder Tests
// ============================================

func TestSessionQuery_WhereID(t *testing.T) {
	initTestDB(t)

	sess := &models.Session{
		SessionID: "sq-test-001",
		GroupName: "group1",
		IsAlive:   true,
		Type:      "beacon",
	}
	if err := Session().Create(sess).Error; err != nil {
		t.Fatalf("Create session failed: %v", err)
	}

	// Find by ID
	result, err := NewSessionQuery().WhereID("sq-test-001").First()
	if err != nil {
		t.Fatalf("WhereID failed: %v", err)
	}
	if result.SessionID != "sq-test-001" {
		t.Errorf("expected session ID 'sq-test-001', got %q", result.SessionID)
	}
}

func TestSessionQuery_WhereAlive(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "alive-1", IsAlive: true})
	Session().Create(&models.Session{SessionID: "dead-1", IsAlive: false})

	alive, err := NewSessionQuery().WhereAlive(true).Find()
	if err != nil {
		t.Fatalf("WhereAlive failed: %v", err)
	}
	for _, s := range alive {
		if !s.IsAlive {
			t.Errorf("expected all sessions to be alive, got dead session %q", s.SessionID)
		}
	}
}

func TestSessionQuery_WhereGroup(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "grp-a", GroupName: "alpha"})
	Session().Create(&models.Session{SessionID: "grp-b", GroupName: "beta"})

	sessions, err := NewSessionQuery().WhereGroup("alpha").Find()
	if err != nil {
		t.Fatalf("WhereGroup failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].GroupName != "alpha" {
		t.Errorf("expected group 'alpha', got %q", sessions[0].GroupName)
	}
}

func TestSessionQuery_Count(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "cnt-1"})
	Session().Create(&models.Session{SessionID: "cnt-2"})
	Session().Create(&models.Session{SessionID: "cnt-3"})

	count, err := NewSessionQuery().Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestSessionQuery_LimitOffset(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "lo-1", GroupName: "a"})
	Session().Create(&models.Session{SessionID: "lo-2", GroupName: "b"})
	Session().Create(&models.Session{SessionID: "lo-3", GroupName: "c"})

	sessions, err := NewSessionQuery().OrderBy("session_id").Limit(2).Find()
	if err != nil {
		t.Fatalf("Limit failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}

	sessions, err = NewSessionQuery().OrderBy("session_id").Offset(1).Limit(1).Find()
	if err != nil {
		t.Fatalf("Offset+Limit failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].SessionID != "lo-2" {
		t.Errorf("expected session 'lo-2', got %q", sessions[0].SessionID)
	}
}

// ============================================
// TaskQuery Builder Tests
// ============================================

func TestTaskQuery_WhereSessionID(t *testing.T) {
	initTestDB(t)

	// Create a session first (FK constraint)
	Session().Create(&models.Session{SessionID: "tq-sess-1"})

	Session().Create(&models.Task{ID: "tq-sess-1-1", SessionID: "tq-sess-1", Seq: 1, Type: "exec"})
	Session().Create(&models.Task{ID: "tq-sess-1-2", SessionID: "tq-sess-1", Seq: 2, Type: "upload"})

	tasks, err := NewTaskQuery().WhereSessionID("tq-sess-1").OrderBySeq().Find()
	if err != nil {
		t.Fatalf("WhereSessionID failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Seq != 1 || tasks[1].Seq != 2 {
		t.Errorf("expected tasks ordered by seq 1,2 got %d,%d", tasks[0].Seq, tasks[1].Seq)
	}
}

func TestTaskQuery_WhereType(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "tq-type-sess"})
	Session().Create(&models.Task{ID: "tq-type-1", SessionID: "tq-type-sess", Seq: 1, Type: "exec"})
	Session().Create(&models.Task{ID: "tq-type-2", SessionID: "tq-type-sess", Seq: 2, Type: "upload"})

	tasks, err := NewTaskQuery().WhereType("exec").Find()
	if err != nil {
		t.Fatalf("WhereType failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Type != "exec" {
		t.Errorf("expected type 'exec', got %q", tasks[0].Type)
	}
}

func TestTaskQuery_Count(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "tq-cnt-sess"})
	Session().Create(&models.Task{ID: "tq-cnt-1", SessionID: "tq-cnt-sess", Seq: 1, Type: "exec"})
	Session().Create(&models.Task{ID: "tq-cnt-2", SessionID: "tq-cnt-sess", Seq: 2, Type: "exec"})

	count, err := NewTaskQuery().WhereSessionID("tq-cnt-sess").Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

// ============================================
// PipelineQuery Builder Tests
// ============================================

func TestPipelineQuery_WhereName(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Pipeline{
		Name:       "pq-tcp-1",
		ListenerId: "listener-1",
		Type:       consts.TCPPipeline,
		PipelineParams: &implanttypes.PipelineParams{
			Tls:        &implanttypes.TlsConfig{},
			Encryption: implanttypes.EncryptionsConfig{},
		},
	})

	result, err := NewPipelineQuery().WhereName("pq-tcp-1").First()
	if err != nil {
		t.Fatalf("WhereName failed: %v", err)
	}
	if result.Name != "pq-tcp-1" {
		t.Errorf("expected name 'pq-tcp-1', got %q", result.Name)
	}
}

func TestPipelineQuery_WhereListenerID(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Pipeline{
		Name: "pq-l-1", ListenerId: "ls-a", Type: consts.TCPPipeline,
		PipelineParams: &implanttypes.PipelineParams{Tls: &implanttypes.TlsConfig{}, Encryption: implanttypes.EncryptionsConfig{}},
	})
	Session().Create(&models.Pipeline{
		Name: "pq-l-2", ListenerId: "ls-b", Type: consts.TCPPipeline,
		PipelineParams: &implanttypes.PipelineParams{Tls: &implanttypes.TlsConfig{}, Encryption: implanttypes.EncryptionsConfig{}},
	})

	pipelines, err := NewPipelineQuery().WhereListenerID("ls-a").Find()
	if err != nil {
		t.Fatalf("WhereListenerID failed: %v", err)
	}
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	if pipelines[0].ListenerId != "ls-a" {
		t.Errorf("expected listener 'ls-a', got %q", pipelines[0].ListenerId)
	}
}

func TestPipelineQuery_WhereTypeAndNotType(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Pipeline{
		Name: "pq-t-tcp", ListenerId: "ls-t", Type: consts.TCPPipeline,
		PipelineParams: &implanttypes.PipelineParams{Tls: &implanttypes.TlsConfig{}, Encryption: implanttypes.EncryptionsConfig{}},
	})
	Session().Create(&models.Pipeline{
		Name: "pq-t-web", ListenerId: "ls-t", Type: consts.WebsitePipeline,
		PipelineParams: &implanttypes.PipelineParams{Tls: &implanttypes.TlsConfig{}},
	})

	// WhereType
	tcpOnly, err := NewPipelineQuery().WhereType(consts.TCPPipeline).Find()
	if err != nil {
		t.Fatalf("WhereType failed: %v", err)
	}
	if len(tcpOnly) != 1 {
		t.Errorf("expected 1 TCP pipeline, got %d", len(tcpOnly))
	}

	// WhereNotType
	nonWeb, err := NewPipelineQuery().WhereNotType(consts.WebsitePipeline).Find()
	if err != nil {
		t.Fatalf("WhereNotType failed: %v", err)
	}
	if len(nonWeb) != 1 {
		t.Errorf("expected 1 non-website pipeline, got %d", len(nonWeb))
	}
}

// ============================================
// List Function Tests
// ============================================

func TestListSessions(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "ls-visible", IsRemoved: false, GroupName: "g1"})
	Session().Create(&models.Session{SessionID: "ls-removed", IsRemoved: true, GroupName: "g2"})

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	// Should only return non-removed sessions
	for _, s := range sessions {
		if s.IsRemoved {
			t.Errorf("ListSessions should not return removed sessions, got %q", s.SessionID)
		}
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestListTasksBySession(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Session{SessionID: "lt-sess-1"})
	Session().Create(&models.Task{ID: "lt-sess-1-1", SessionID: "lt-sess-1", Seq: 2, Type: "exec"})
	Session().Create(&models.Task{ID: "lt-sess-1-2", SessionID: "lt-sess-1", Seq: 1, Type: "upload"})

	tasks, err := ListTasksBySession("lt-sess-1")
	if err != nil {
		t.Fatalf("ListTasksBySession failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	// Should be ordered by seq ASC
	if tasks[0].Seq != 1 {
		t.Errorf("expected first task seq=1, got %d", tasks[0].Seq)
	}
}

func TestListPipelinesByListener(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Pipeline{
		Name: "lp-tcp", ListenerId: "lp-ls", Type: consts.TCPPipeline,
		PipelineParams: &implanttypes.PipelineParams{Tls: &implanttypes.TlsConfig{}, Encryption: implanttypes.EncryptionsConfig{}},
	})
	Session().Create(&models.Pipeline{
		Name: "lp-web", ListenerId: "lp-ls", Type: consts.WebsitePipeline,
		PipelineParams: &implanttypes.PipelineParams{Tls: &implanttypes.TlsConfig{}},
	})

	// ListPipelinesByListener should exclude website pipelines
	pipelines, err := ListPipelinesByListener("lp-ls")
	if err != nil {
		t.Fatalf("ListPipelinesByListener failed: %v", err)
	}
	if len(pipelines) != 1 {
		t.Errorf("expected 1 non-website pipeline, got %d", len(pipelines))
	}

	// ListWebsitesByListener should only return website pipelines
	websites, err := ListWebsitesByListener("lp-ls")
	if err != nil {
		t.Fatalf("ListWebsitesByListener failed: %v", err)
	}
	if len(websites) != 1 {
		t.Errorf("expected 1 website pipeline, got %d", len(websites))
	}
}

func TestListPipelinesByListener_EmptyListenerID(t *testing.T) {
	initTestDB(t)

	Session().Create(&models.Pipeline{
		Name: "lp-all-1", ListenerId: "ls-x", Type: consts.TCPPipeline,
		PipelineParams: &implanttypes.PipelineParams{Tls: &implanttypes.TlsConfig{}, Encryption: implanttypes.EncryptionsConfig{}},
	})
	Session().Create(&models.Pipeline{
		Name: "lp-all-2", ListenerId: "ls-y", Type: consts.HTTPPipeline,
		PipelineParams: &implanttypes.PipelineParams{Tls: &implanttypes.TlsConfig{}, Encryption: implanttypes.EncryptionsConfig{}},
	})

	// Empty listenerID should return all non-website pipelines
	pipelines, err := ListPipelinesByListener("")
	if err != nil {
		t.Fatalf("ListPipelinesByListener('') failed: %v", err)
	}
	if len(pipelines) != 2 {
		t.Errorf("expected 2 pipelines, got %d", len(pipelines))
	}
}

// ============================================
// Internal Helper Tests
// ============================================

func TestSaveWithOmitEmpty(t *testing.T) {
	initTestDB(t)

	sess := &models.Session{
		SessionID:   "omit-test-1",
		ProfileName: "",
		GroupName:   "test-group",
	}
	// Create first
	if err := Session().Create(sess).Error; err != nil {
		t.Fatalf("Create session failed: %v", err)
	}

	// Save with omit should not fail on empty FK
	sess.GroupName = "updated-group"
	if err := saveWithOmitEmpty(sess, map[string]string{"profile_name": sess.ProfileName}); err != nil {
		t.Fatalf("saveWithOmitEmpty failed: %v", err)
	}

	// Verify update
	var result models.Session
	Session().Where("session_id = ?", "omit-test-1").First(&result)
	if result.GroupName != "updated-group" {
		t.Errorf("expected 'updated-group', got %q", result.GroupName)
	}
}

func TestCreateWithOmitEmpty(t *testing.T) {
	initTestDB(t)

	sess := &models.Session{
		SessionID:   "create-omit-1",
		ProfileName: "",
		GroupName:   "grp",
	}
	if err := createWithOmitEmpty(sess, map[string]string{"profile_name": sess.ProfileName}); err != nil {
		t.Fatalf("createWithOmitEmpty failed: %v", err)
	}

	var result models.Session
	if err := Session().Where("session_id = ?", "create-omit-1").First(&result).Error; err != nil {
		t.Fatalf("Failed to find created session: %v", err)
	}
	if result.GroupName != "grp" {
		t.Errorf("expected 'grp', got %q", result.GroupName)
	}
}
