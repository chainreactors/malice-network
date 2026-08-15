package rpc

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestListSessionsDefaultsAndPreservesFullSession(t *testing.T) {
	newRPCTestEnv(t)

	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 12; i++ {
		sessionID := "session-" + string(rune('a'+i))
		fixture := &models.Session{
			SessionID: sessionID,
			IsAlive:   i != 11,
			Data:      newSessionListRPCContext(),
		}
		if i == 10 {
			fixture.Note = "full payload"
			fixture.Data = &client.SessionContext{
				SessionInfo: &client.SessionInfo{
					Os:          &implantpb.Os{Name: "linux", Arch: "amd64"},
					Process:     &implantpb.Process{Name: "agent"},
					Expression:  "*/5 * * * *",
					Jitter:      0.25,
					IsPrivilege: true,
					Filepath:    "/opt/agent",
					WorkDir:     "/tmp",
					ProxyURL:    "socks5://127.0.0.1:1080",
					Locale:      "en-US",
				},
				KeyPair: &clientpb.KeyPair{
					PublicKey:  "public-key",
					PrivateKey: "private-key",
				},
				Modules: []string{"exec", "upload"},
				Addons:  []*implantpb.Addon{{Name: "test-addon", Type: "module"}},
				Argue:   map[string]string{"process": "explorer.exe"},
				Any:     map[string]interface{}{},
			}
		}
		seedSessionListRPCFixture(t, fixture, createdAt.Add(time.Duration(i)*time.Minute))
	}

	response, err := (&Server{}).ListSessions(context.Background(), &clientpb.ListSessionsRequest{})
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if response.GetPage() != 1 || response.GetPageSize() != 10 {
		t.Fatalf("page metadata = (%d, %d), want (1, 10)", response.GetPage(), response.GetPageSize())
	}
	if response.GetFilteredTotal() != 12 {
		t.Fatalf("filtered_total = %d, want 12", response.GetFilteredTotal())
	}
	if len(response.GetSessions()) != 10 {
		t.Fatalf("sessions = %d, want 10", len(response.GetSessions()))
	}

	wantIDs := []string{
		"session-k", "session-j", "session-i", "session-h", "session-g",
		"session-f", "session-e", "session-d", "session-c", "session-b",
	}
	if got := sessionListRPCIDs(response.GetSessions()); !reflect.DeepEqual(got, wantIDs) {
		t.Fatalf("session IDs = %v, want %v", got, wantIDs)
	}

	full := response.GetSessions()[0]
	if full.GetData() == "" {
		t.Fatal("full session data was omitted")
	}
	if !reflect.DeepEqual(full.GetModules(), []string{"exec", "upload"}) {
		t.Fatalf("modules = %v, want [exec upload]", full.GetModules())
	}
	if len(full.GetAddons()) != 1 || full.GetAddons()[0].GetName() != "test-addon" {
		t.Fatalf("addons = %#v, want test-addon", full.GetAddons())
	}
	if full.GetKeyPair().GetPublicKey() != "public-key" || full.GetKeyPair().GetPrivateKey() != "private-key" {
		t.Fatalf("key pair = %#v, want full key pair", full.GetKeyPair())
	}
}

func TestListSessionsFiltersSearchesAndSorts(t *testing.T) {
	newRPCTestEnv(t)

	createdAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fixtures := []*models.Session{
		{SessionID: "red-late", GroupName: "red", Note: "Needle", IsAlive: true, LastCheckin: 30},
		{SessionID: "red-early", GroupName: "red", Target: "needle-host", IsAlive: true, LastCheckin: 10},
		{SessionID: "red-offline", GroupName: "red", Note: "needle", IsAlive: false, LastCheckin: 20},
		{SessionID: "blue-alive", GroupName: "blue", Note: "needle", IsAlive: true, LastCheckin: 5},
		{SessionID: "red-other", GroupName: "red", Note: "other", IsAlive: true, LastCheckin: 1},
		{SessionID: "red-removed", GroupName: "red", Note: "needle", IsAlive: true, IsRemoved: true},
	}
	for _, fixture := range fixtures {
		fixture.Data = newSessionListRPCContext()
		seedSessionListRPCFixture(t, fixture, createdAt)
	}

	response, err := (&Server{}).ListSessions(context.Background(), &clientpb.ListSessionsRequest{
		Page:          1,
		PageSize:      20,
		Status:        clientpb.SessionListStatus_SESSION_LIST_STATUS_ALIVE,
		Group:         "red",
		Search:        "NEEDLE",
		SortField:     clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_LAST_CHECKIN,
		SortDirection: clientpb.SessionListSortDirection_SESSION_LIST_SORT_DIRECTION_ASC,
	})
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if response.GetFilteredTotal() != 2 {
		t.Fatalf("filtered_total = %d, want 2", response.GetFilteredTotal())
	}
	if stats := response.GetStats(); stats.GetTotal() != 3 || stats.GetAlive() != 2 || stats.GetOffline() != 1 {
		t.Fatalf("stats = %+v, want total=3 alive=2 offline=1", stats)
	}
	if got, want := sessionListRPCIDs(response.GetSessions()), []string{"red-early", "red-late"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("session IDs = %v, want %v", got, want)
	}
}

func TestListSessionsAcceptsMaximumPageSize(t *testing.T) {
	newRPCTestEnv(t)
	seedSessionListRPCFixture(t, &models.Session{
		SessionID: "maximum-page-size",
		Data:      newSessionListRPCContext(),
	}, time.Now())

	response, err := (&Server{}).ListSessions(context.Background(), &clientpb.ListSessionsRequest{
		Page:     1,
		PageSize: 5000,
	})
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if response.GetPageSize() != 5000 {
		t.Fatalf("page_size = %d, want 5000", response.GetPageSize())
	}
}

func TestListSessionsRejectsInvalidArguments(t *testing.T) {
	newRPCTestEnv(t)

	tests := []struct {
		name    string
		request *clientpb.ListSessionsRequest
	}{
		{name: "nil request"},
		{name: "negative page", request: &clientpb.ListSessionsRequest{Page: -1}},
		{name: "negative page size", request: &clientpb.ListSessionsRequest{PageSize: -1}},
		{name: "page size above maximum", request: &clientpb.ListSessionsRequest{PageSize: 5001}},
		{name: "unknown status", request: &clientpb.ListSessionsRequest{Status: clientpb.SessionListStatus(99)}},
		{name: "unknown sort field", request: &clientpb.ListSessionsRequest{SortField: clientpb.SessionListSortField(99)}},
		{
			name: "missing explicit sort direction",
			request: &clientpb.ListSessionsRequest{
				SortField: clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_CREATED_AT,
			},
		},
		{
			name: "unknown sort direction",
			request: &clientpb.ListSessionsRequest{
				SortField:     clientpb.SessionListSortField_SESSION_LIST_SORT_FIELD_CREATED_AT,
				SortDirection: clientpb.SessionListSortDirection(99),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := (&Server{}).ListSessions(context.Background(), test.request)
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("error = %v, code = %s, want %s", err, status.Code(err), codes.InvalidArgument)
			}
		})
	}
}

func TestSessionStatsAndGroups(t *testing.T) {
	newRPCTestEnv(t)

	createdAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	fixtures := []*models.Session{
		{SessionID: "red-alive", GroupName: "red", Note: "needle", IsAlive: true},
		{SessionID: "red-offline", GroupName: "red", Target: "needle-host", IsAlive: false},
		{SessionID: "blue-alive", GroupName: "blue", IsAlive: true},
		{SessionID: "empty-group", IsAlive: false},
		{SessionID: "removed-group", GroupName: "hidden", Note: "needle", IsAlive: true, IsRemoved: true},
	}
	for _, fixture := range fixtures {
		fixture.Data = newSessionListRPCContext()
		seedSessionListRPCFixture(t, fixture, createdAt)
	}

	stats, err := (&Server{}).GetSessionStats(context.Background(), &clientpb.SessionStatsRequest{
		Group:  "red",
		Search: "NEEDLE",
	})
	if err != nil {
		t.Fatalf("GetSessionStats failed: %v", err)
	}
	if stats.GetTotal() != 2 || stats.GetAlive() != 1 || stats.GetOffline() != 1 {
		t.Fatalf("stats = %+v, want total=2 alive=1 offline=1", stats)
	}

	groups, err := (&Server{}).ListSessionGroups(context.Background(), &clientpb.Empty{})
	if err != nil {
		t.Fatalf("ListSessionGroups failed: %v", err)
	}
	if want := []string{"blue", "red"}; !reflect.DeepEqual(groups.GetGroups(), want) {
		t.Fatalf("groups = %v, want %v", groups.GetGroups(), want)
	}
}

func TestGetSessionTrendReturnsTwentyFourAscendingBuckets(t *testing.T) {
	newRPCTestEnv(t)

	beforeCallHour := time.Now().UTC().Truncate(time.Hour)
	trend, err := (&Server{}).GetSessionTrend(context.Background(), &clientpb.Empty{})
	if err != nil {
		t.Fatalf("GetSessionTrend failed: %v", err)
	}
	afterCallHour := time.Now().UTC().Truncate(time.Hour)

	points := trend.GetPoints()
	if len(points) != db.SessionTrendBucketCount {
		t.Fatalf("point count = %d, want %d", len(points), db.SessionTrendBucketCount)
	}
	lastBucketStart := time.Unix(points[len(points)-1].GetBucketStartUnix(), 0).UTC()
	if !lastBucketStart.Equal(beforeCallHour) && !lastBucketStart.Equal(afterCallHour) {
		t.Fatalf(
			"last bucket = %s, want invocation hour %s or %s",
			lastBucketStart,
			beforeCallHour,
			afterCallHour,
		)
	}
	for i, point := range points {
		wantStart := lastBucketStart.Add(time.Duration(i-len(points)+1) * time.Hour).Unix()
		if point.GetBucketStartUnix() != wantStart {
			t.Fatalf("point %d start = %d, want %d", i, point.GetBucketStartUnix(), wantStart)
		}
		if point.GetCount() != 0 {
			t.Fatalf("point %d count = %d, want 0", i, point.GetCount())
		}
	}

	if _, err := (&Server{}).GetSessionTrend(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil request error = %v, code = %s, want %s", err, status.Code(err), codes.InvalidArgument)
	}
}

func TestSessionListRPCsMapDatabaseErrorsToInternal(t *testing.T) {
	tests := []struct {
		name string
		call func(*Server) error
	}{
		{
			name: "list sessions",
			call: func(server *Server) error {
				_, err := server.ListSessions(context.Background(), &clientpb.ListSessionsRequest{})
				return err
			},
		},
		{
			name: "session stats",
			call: func(server *Server) error {
				_, err := server.GetSessionStats(context.Background(), &clientpb.SessionStatsRequest{})
				return err
			},
		},
		{
			name: "session groups",
			call: func(server *Server) error {
				_, err := server.ListSessionGroups(context.Background(), &clientpb.Empty{})
				return err
			},
		},
		{
			name: "session trend",
			call: func(server *Server) error {
				_, err := server.GetSessionTrend(context.Background(), &clientpb.Empty{})
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newRPCTestEnv(t)
			sqlDB, err := db.Client.DB()
			if err != nil {
				t.Fatalf("open underlying database: %v", err)
			}
			if err := sqlDB.Close(); err != nil {
				t.Fatalf("close underlying database: %v", err)
			}

			err = test.call(&Server{})
			if status.Code(err) != codes.Internal {
				t.Fatalf("error = %v, code = %s, want %s", err, status.Code(err), codes.Internal)
			}
		})
	}
}

func TestSessionListRPCsPropagateCancellation(t *testing.T) {
	newRPCTestEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "list sessions",
			call: func() error {
				_, err := (&Server{}).ListSessions(ctx, &clientpb.ListSessionsRequest{})
				return err
			},
		},
		{
			name: "session stats",
			call: func() error {
				_, err := (&Server{}).GetSessionStats(ctx, &clientpb.SessionStatsRequest{})
				return err
			},
		},
		{
			name: "session groups",
			call: func() error {
				_, err := (&Server{}).ListSessionGroups(ctx, &clientpb.Empty{})
				return err
			},
		},
		{
			name: "session trend",
			call: func() error {
				_, err := (&Server{}).GetSessionTrend(ctx, &clientpb.Empty{})
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); status.Code(err) != codes.Canceled {
				t.Fatalf("error = %v, code = %s, want %s", err, status.Code(err), codes.Canceled)
			}
		})
	}
}

func seedSessionListRPCFixture(t testing.TB, session *models.Session, createdAt time.Time) {
	t.Helper()
	if err := db.Session().Create(session).Error; err != nil {
		t.Fatalf("create session %q: %v", session.SessionID, err)
	}
	if err := db.Session().Exec(
		"UPDATE sessions SET created_at = ? WHERE session_id = ?",
		createdAt,
		session.SessionID,
	).Error; err != nil {
		t.Fatalf("set created_at for session %q: %v", session.SessionID, err)
	}
}

func newSessionListRPCContext() *client.SessionContext {
	return &client.SessionContext{
		SessionInfo: &client.SessionInfo{
			Os:      &implantpb.Os{},
			Process: &implantpb.Process{},
		},
		Argue: map[string]string{},
		Any:   map[string]interface{}{},
	}
}

func sessionListRPCIDs(sessions []*clientpb.Session) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.GetSessionId())
	}
	return ids
}
