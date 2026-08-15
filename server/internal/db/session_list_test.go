package db

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func TestListSessionsPage_DefaultOrderPaginationAndTotal(t *testing.T) {
	initTestDB(t)

	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	createSessionListFixture(t, &models.Session{SessionID: "alive-a", IsAlive: true}, newer)
	createSessionListFixture(t, &models.Session{SessionID: "alive-b", IsAlive: true}, newer)
	createSessionListFixture(t, &models.Session{SessionID: "alive-old", IsAlive: true}, older)
	createSessionListFixture(t, &models.Session{SessionID: "offline-new", IsAlive: false}, newer)
	createSessionListFixture(t, &models.Session{SessionID: "offline-old", IsAlive: false}, older)
	createSessionListFixture(t, &models.Session{SessionID: "removed", IsAlive: true, IsRemoved: true}, newer.Add(time.Hour))

	first, err := ListSessionsPage(SessionListOptions{
		Page:     1,
		PageSize: 3,
		Status:   SessionListStatusAll,
	})
	if err != nil {
		t.Fatalf("ListSessionsPage first page failed: %v", err)
	}
	if first.Total != 5 {
		t.Fatalf("total = %d, want 5", first.Total)
	}
	if first.Page != 1 || first.PageSize != 3 {
		t.Fatalf("page metadata = (%d, %d), want (1, 3)", first.Page, first.PageSize)
	}
	assertSessionListIDs(t, first.Sessions, "alive-a", "alive-b", "alive-old")

	second, err := ListSessionsPage(SessionListOptions{
		Page:     2,
		PageSize: 3,
	})
	if err != nil {
		t.Fatalf("ListSessionsPage second page failed: %v", err)
	}
	if second.Total != 5 {
		t.Fatalf("total = %d, want 5", second.Total)
	}
	assertSessionListIDs(t, second.Sessions, "offline-new", "offline-old")

	beyond, err := ListSessionsPage(SessionListOptions{Page: 3, PageSize: 3})
	if err != nil {
		t.Fatalf("ListSessionsPage beyond last page failed: %v", err)
	}
	if beyond.Sessions == nil || len(beyond.Sessions) != 0 {
		t.Fatalf("sessions beyond last page = %#v, want non-nil empty slice", beyond.Sessions)
	}

	defaults, err := ListSessionsPage(SessionListOptions{})
	if err != nil {
		t.Fatalf("ListSessionsPage defaults failed: %v", err)
	}
	if defaults.Page != DefaultSessionListPage || defaults.PageSize != DefaultSessionListPageSize {
		t.Fatalf(
			"default page metadata = (%d, %d), want (%d, %d)",
			defaults.Page,
			defaults.PageSize,
			DefaultSessionListPage,
			DefaultSessionListPageSize,
		)
	}
}

func TestListSessionsPage_Filters(t *testing.T) {
	initTestDB(t)

	createdAt := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	createSessionListFixture(t, &models.Session{SessionID: "red-alive-a", GroupName: "red", IsAlive: true}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "red-alive-b", GroupName: "red", IsAlive: true}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "red-offline", GroupName: "red", IsAlive: false}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "blue-alive", GroupName: "blue", IsAlive: true}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "red-removed", GroupName: "red", IsAlive: true, IsRemoved: true}, createdAt)

	tests := []struct {
		name   string
		status SessionListStatus
		group  string
		want   []string
	}{
		{name: "all", status: SessionListStatusAll, want: []string{"blue-alive", "red-alive-a", "red-alive-b", "red-offline"}},
		{name: "zero status defaults to all", want: []string{"blue-alive", "red-alive-a", "red-alive-b", "red-offline"}},
		{name: "alive", status: SessionListStatusAlive, want: []string{"blue-alive", "red-alive-a", "red-alive-b"}},
		{name: "offline", status: SessionListStatusOffline, want: []string{"red-offline"}},
		{name: "group", status: SessionListStatusAll, group: "red", want: []string{"red-alive-a", "red-alive-b", "red-offline"}},
		{name: "combined", status: SessionListStatusAlive, group: "red", want: []string{"red-alive-a", "red-alive-b"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ListSessionsPage(SessionListOptions{
				Page:     1,
				PageSize: 20,
				Status:   test.status,
				Group:    test.group,
			})
			if err != nil {
				t.Fatalf("ListSessionsPage failed: %v", err)
			}
			if result.Total != int64(len(test.want)) {
				t.Fatalf("total = %d, want %d", result.Total, len(test.want))
			}
			assertSessionListIDs(t, result.Sessions, test.want...)
		})
	}
}

func TestListSessionsPage_StatusTotalIsDerivedFromUnfilteredStats(t *testing.T) {
	initTestDB(t)

	createdAt := time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC)
	createSessionListFixture(t, &models.Session{SessionID: "matched-alive-a", GroupName: "red", Note: "needle", IsAlive: true}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "matched-alive-b", GroupName: "red", Target: "needle-host", IsAlive: true}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "matched-offline", GroupName: "red", Note: "needle", IsAlive: false}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "wrong-group", GroupName: "blue", Note: "needle", IsAlive: false}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "wrong-search", GroupName: "red", Note: "other", IsAlive: false}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "removed", GroupName: "red", Note: "needle", IsAlive: false, IsRemoved: true}, createdAt)

	result, err := ListSessionsPage(SessionListOptions{
		Page:     1,
		PageSize: 10,
		Status:   SessionListStatusOffline,
		Group:    "red",
		Search:   "needle",
	})
	if err != nil {
		t.Fatalf("ListSessionsPage failed: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("filtered total = %d, want 1", result.Total)
	}
	if want := (SessionStats{Total: 3, Alive: 2, Offline: 1}); result.Stats != want {
		t.Fatalf("stats = %+v, want %+v", result.Stats, want)
	}
	assertSessionListIDs(t, result.Sessions, "matched-offline")
}

func TestListSessionsPage_SearchesStableFieldsAndData(t *testing.T) {
	initTestDB(t)

	createdAt := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	createSessionListFixture(t, &models.Session{
		SessionID:   "search-id-token",
		Note:        "NoteToken",
		GroupName:   "GroupToken",
		Target:      "TargetToken",
		Type:        "TypeToken",
		PipelineID:  "PipelineToken",
		ListenerID:  "ListenerToken",
		ProfileName: "ProfileToken",
		Data: &client.SessionContext{
			SessionInfo: &client.SessionInfo{Locale: "DataToken"},
		},
	}, createdAt)
	createSessionListFixture(t, &models.Session{
		SessionID: "literal-match",
		Note:      "literal%_!token",
	}, createdAt)
	createSessionListFixture(t, &models.Session{
		SessionID: "wildcard-distractor",
		Note:      "literalABCxtoken",
	}, createdAt)
	createSessionListFixture(t, &models.Session{
		SessionID: "removed-search",
		Note:      "NoteToken",
		IsRemoved: true,
	}, createdAt)

	tests := []struct {
		name   string
		search string
		wantID string
	}{
		{name: "session id", search: "ID-TOKEN", wantID: "search-id-token"},
		{name: "note", search: "notetoken", wantID: "search-id-token"},
		{name: "group", search: "grouptoken", wantID: "search-id-token"},
		{name: "target", search: "targettoken", wantID: "search-id-token"},
		{name: "type", search: "typetoken", wantID: "search-id-token"},
		{name: "pipeline", search: "pipelinetoken", wantID: "search-id-token"},
		{name: "listener", search: "listenertoken", wantID: "search-id-token"},
		{name: "profile", search: "profiletoken", wantID: "search-id-token"},
		{name: "data string", search: "datatoken", wantID: "search-id-token"},
		{name: "literal LIKE metacharacters", search: "literal%_!token", wantID: "literal-match"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ListSessionsPage(SessionListOptions{
				Page:     1,
				PageSize: 10,
				Search:   test.search,
			})
			if err != nil {
				t.Fatalf("ListSessionsPage failed: %v", err)
			}
			if result.Total != 1 {
				t.Fatalf("total = %d, want 1", result.Total)
			}
			assertSessionListIDs(t, result.Sessions, test.wantID)
		})
	}
}

func TestSessionListSearchPredicateUsesDialectCaseFolding(t *testing.T) {
	if got := sessionListSearchPredicate("sqlite", "note"); got != "note LIKE ? ESCAPE '!'" {
		t.Fatalf("SQLite predicate = %q", got)
	}
	if got := sessionListSearchPredicate("postgres", "note"); got != "LOWER(COALESCE(note, '')) LIKE ? ESCAPE '!'" {
		t.Fatalf("PostgreSQL predicate = %q", got)
	}
}

func TestListSessionsPage_ExplicitSortUsesStableTieBreaker(t *testing.T) {
	initTestDB(t)

	createdAt := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	createSessionListFixture(t, &models.Session{SessionID: "sort-a", GroupName: "beta", LastCheckin: 10}, createdAt.Add(time.Hour))
	createSessionListFixture(t, &models.Session{SessionID: "sort-b", GroupName: "beta", LastCheckin: 20, IsAlive: true}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "sort-c", GroupName: "alpha", LastCheckin: 30}, createdAt)

	tests := []struct {
		name string
		sort SessionListSort
		want []string
	}{
		{
			name: "group ascending with operational tie-break",
			sort: SessionListSort{Field: SessionListSortFieldGroup, Direction: SessionListSortAscending},
			want: []string{"sort-c", "sort-b", "sort-a"},
		},
		{
			name: "group descending with operational tie-break",
			sort: SessionListSort{Field: SessionListSortFieldGroup, Direction: SessionListSortDescending},
			want: []string{"sort-b", "sort-a", "sort-c"},
		},
		{
			name: "last checkin descending",
			sort: SessionListSort{Field: SessionListSortFieldLastCheckin, Direction: SessionListSortDescending},
			want: []string{"sort-c", "sort-b", "sort-a"},
		},
		{
			name: "session id descending",
			sort: SessionListSort{Field: SessionListSortFieldSessionID, Direction: SessionListSortDescending},
			want: []string{"sort-c", "sort-b", "sort-a"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ListSessionsPage(SessionListOptions{
				Page:     1,
				PageSize: 10,
				Sort:     &test.sort,
			})
			if err != nil {
				t.Fatalf("ListSessionsPage failed: %v", err)
			}
			assertSessionListIDs(t, result.Sessions, test.want...)
		})
	}
}

func TestListSessionsPage_StatusAscendingReversesOperationalOrder(t *testing.T) {
	initTestDB(t)

	older := time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	createSessionListFixture(t, &models.Session{SessionID: "offline-old", IsAlive: false}, older)
	createSessionListFixture(t, &models.Session{SessionID: "offline-new", IsAlive: false}, newer)
	createSessionListFixture(t, &models.Session{SessionID: "alive-old", IsAlive: true}, older)
	createSessionListFixture(t, &models.Session{SessionID: "alive-new-a", IsAlive: true}, newer)
	createSessionListFixture(t, &models.Session{SessionID: "alive-new-b", IsAlive: true}, newer)

	sort := SessionListSort{
		Field:     SessionListSortFieldIsAlive,
		Direction: SessionListSortAscending,
	}
	result, err := ListSessionsPage(SessionListOptions{Page: 1, PageSize: 10, Sort: &sort})
	if err != nil {
		t.Fatalf("ListSessionsPage failed: %v", err)
	}
	assertSessionListIDs(
		t,
		result.Sessions,
		"offline-old",
		"offline-new",
		"alive-old",
		"alive-new-b",
		"alive-new-a",
	)
}

func TestListSessionsPage_ValidatesOptions(t *testing.T) {
	valid := SessionListOptions{Page: 1, PageSize: 10}
	maliciousSort := SessionListSort{Field: SessionListSortField("created_at; DROP TABLE sessions"), Direction: SessionListSortAscending}
	invalidDirection := SessionListSort{Field: SessionListSortFieldCreatedAt, Direction: SessionListSortDirection("sideways")}

	tests := []struct {
		name    string
		options SessionListOptions
	}{
		{name: "page below zero", options: SessionListOptions{Page: -1, PageSize: 10}},
		{name: "page size below zero", options: SessionListOptions{Page: 1, PageSize: -1}},
		{name: "page size above maximum", options: SessionListOptions{Page: 1, PageSize: MaxSessionListPageSize + 1}},
		{name: "unsupported status", options: SessionListOptions{Page: 1, PageSize: 10, Status: SessionListStatus("unknown")}},
		{name: "sort field is not free SQL", options: SessionListOptions{Page: 1, PageSize: 10, Sort: &maliciousSort}},
		{name: "unsupported sort direction", options: SessionListOptions{Page: 1, PageSize: 10, Sort: &invalidDirection}},
		{name: "offset overflow", options: SessionListOptions{Page: math.MaxInt, PageSize: 2}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ListSessionsPage(test.options)
			if !errors.Is(err, ErrInvalidSessionListOptions) {
				t.Fatalf("error = %v, want ErrInvalidSessionListOptions", err)
			}
		})
	}

	if _, _, err := normalizeSessionListOptions(valid); err != nil {
		t.Fatalf("valid options failed validation: %v", err)
	}
}

func TestGetSessionStats(t *testing.T) {
	initTestDB(t)

	createdAt := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	createSessionListFixture(t, &models.Session{
		SessionID: "stats-red-alive",
		GroupName: "red",
		IsAlive:   true,
		Note:      "needle",
	}, createdAt)
	createSessionListFixture(t, &models.Session{
		SessionID: "stats-red-offline",
		GroupName: "red",
		IsAlive:   false,
		Target:    "needle",
	}, createdAt)
	createSessionListFixture(t, &models.Session{
		SessionID: "stats-blue-alive",
		GroupName: "blue",
		IsAlive:   true,
	}, createdAt)
	createSessionListFixture(t, &models.Session{
		SessionID: "stats-removed",
		GroupName: "red",
		IsAlive:   true,
		Note:      "needle",
		IsRemoved: true,
	}, createdAt)

	tests := []struct {
		name    string
		options SessionStatsOptions
		want    SessionStats
	}{
		{
			name: "all visible sessions",
			want: SessionStats{Total: 3, Alive: 2, Offline: 1},
		},
		{
			name:    "group",
			options: SessionStatsOptions{Group: "red"},
			want:    SessionStats{Total: 2, Alive: 1, Offline: 1},
		},
		{
			name:    "search",
			options: SessionStatsOptions{Search: "NEEDLE"},
			want:    SessionStats{Total: 2, Alive: 1, Offline: 1},
		},
		{
			name:    "combined filters with no result",
			options: SessionStatsOptions{Group: "blue", Search: "needle"},
			want:    SessionStats{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetSessionStats(test.options)
			if err != nil {
				t.Fatalf("GetSessionStats failed: %v", err)
			}
			if *got != test.want {
				t.Fatalf("stats = %+v, want %+v", *got, test.want)
			}
		})
	}
}

func TestSessionListQueriesRespectCanceledContext(t *testing.T) {
	initTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "list",
			call: func() error {
				_, err := ListSessionsPageContext(ctx, SessionListOptions{})
				return err
			},
		},
		{
			name: "stats",
			call: func() error {
				_, err := GetSessionStatsContext(ctx, SessionStatsOptions{})
				return err
			},
		},
		{
			name: "groups",
			call: func() error {
				_, err := ListSessionGroupsContext(ctx)
				return err
			},
		},
		{
			name: "trend",
			call: func() error {
				_, err := GetSessionTrendContext(ctx, time.Now())
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
		})
	}
}

func TestListSessionGroups(t *testing.T) {
	initTestDB(t)

	createdAt := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	createSessionListFixture(t, &models.Session{SessionID: "group-beta", GroupName: "beta"}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "group-alpha-a", GroupName: "alpha"}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "group-alpha-b", GroupName: "alpha"}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "group-empty"}, createdAt)
	createSessionListFixture(t, &models.Session{SessionID: "group-removed", GroupName: "gamma", IsRemoved: true}, createdAt)

	groups, err := ListSessionGroups()
	if err != nil {
		t.Fatalf("ListSessionGroups failed: %v", err)
	}
	if !reflect.DeepEqual(groups, []string{"alpha", "beta"}) {
		t.Fatalf("groups = %v, want [alpha beta]", groups)
	}
}

func TestListSessionsPage_DecodesOnlyCurrentPage(t *testing.T) {
	initTestDB(t)

	older := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	createSessionListFixture(t, &models.Session{
		SessionID: "valid-current-page",
		Data: &client.SessionContext{
			SessionInfo: &client.SessionInfo{Locale: "decoded"},
		},
	}, newer)
	createSessionListFixture(t, &models.Session{
		SessionID:  "invalid-later-page",
		DataString: "{invalid-json",
	}, older)

	sort := SessionListSort{Field: SessionListSortFieldCreatedAt, Direction: SessionListSortDescending}
	first, err := ListSessionsPage(SessionListOptions{Page: 1, PageSize: 1, Sort: &sort})
	if err != nil {
		t.Fatalf("first page should not decode later rows: %v", err)
	}
	if first.Total != 2 {
		t.Fatalf("total = %d, want 2", first.Total)
	}
	assertSessionListIDs(t, first.Sessions, "valid-current-page")
	if first.Sessions[0].Data == nil || first.Sessions[0].Data.Locale != "decoded" {
		t.Fatalf("current page data was not decoded: %#v", first.Sessions[0].Data)
	}

	if _, err := ListSessionsPage(SessionListOptions{Page: 2, PageSize: 1, Sort: &sort}); err == nil {
		t.Fatal("second page should report its invalid session data")
	}
}

func createSessionListFixture(t *testing.T, session *models.Session, createdAt time.Time) {
	t.Helper()
	if err := Session().Create(session).Error; err != nil {
		t.Fatalf("create session %q: %v", session.SessionID, err)
	}
	if err := Session().Exec(
		"UPDATE sessions SET created_at = ? WHERE session_id = ?",
		createdAt,
		session.SessionID,
	).Error; err != nil {
		t.Fatalf("set created_at for session %q: %v", session.SessionID, err)
	}
}

func assertSessionListIDs(t *testing.T, sessions Sessions, want ...string) {
	t.Helper()
	got := make([]string, 0, len(sessions))
	for _, session := range sessions {
		got = append(got, session.SessionID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("session IDs = %v, want %v", got, want)
	}
}
