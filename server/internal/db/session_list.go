package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DefaultSessionListPage     = 1
	DefaultSessionListPageSize = 10
	MaxSessionListPageSize     = 5000
)

var ErrInvalidSessionListOptions = errors.New("invalid session list options")

type SessionListStatus string

const (
	SessionListStatusAll     SessionListStatus = "all"
	SessionListStatusAlive   SessionListStatus = "alive"
	SessionListStatusOffline SessionListStatus = "offline"
)

type SessionListSortField string

const (
	SessionListSortFieldSessionID   SessionListSortField = "session_id"
	SessionListSortFieldRawID       SessionListSortField = "raw_id"
	SessionListSortFieldCreatedAt   SessionListSortField = "created_at"
	SessionListSortFieldNote        SessionListSortField = "note"
	SessionListSortFieldGroup       SessionListSortField = "group_name"
	SessionListSortFieldTarget      SessionListSortField = "target"
	SessionListSortFieldInitialized SessionListSortField = "initialized"
	SessionListSortFieldType        SessionListSortField = "type"
	SessionListSortFieldPipelineID  SessionListSortField = "pipeline_id"
	SessionListSortFieldListenerID  SessionListSortField = "listener_id"
	SessionListSortFieldIsAlive     SessionListSortField = "is_alive"
	SessionListSortFieldLastCheckin SessionListSortField = "last_checkin"
	SessionListSortFieldProfileName SessionListSortField = "profile_name"
)

type SessionListSortDirection string

const (
	SessionListSortAscending  SessionListSortDirection = "asc"
	SessionListSortDescending SessionListSortDirection = "desc"
)

type SessionListSort struct {
	Field     SessionListSortField
	Direction SessionListSortDirection
}

type SessionListOptions struct {
	Page     int
	PageSize int
	Status   SessionListStatus
	Group    string
	Search   string
	Sort     *SessionListSort
}

type SessionListResult struct {
	Sessions Sessions
	Total    int64
	Stats    SessionStats
	Page     int
	PageSize int
}

type SessionStatsOptions struct {
	Group  string
	Search string
}

type SessionStats struct {
	Total   int64
	Alive   int64
	Offline int64
}

var sessionListSortColumns = map[SessionListSortField]string{
	SessionListSortFieldSessionID:   "session_id",
	SessionListSortFieldRawID:       "raw_id",
	SessionListSortFieldCreatedAt:   "created_at",
	SessionListSortFieldNote:        "note",
	SessionListSortFieldGroup:       "group_name",
	SessionListSortFieldTarget:      "target",
	SessionListSortFieldInitialized: "initialized",
	SessionListSortFieldType:        "type",
	SessionListSortFieldPipelineID:  "pipeline_id",
	SessionListSortFieldListenerID:  "listener_id",
	SessionListSortFieldIsAlive:     "is_alive",
	SessionListSortFieldLastCheckin: "last_checkin",
	SessionListSortFieldProfileName: "profile_name",
}

var sessionListSearchColumns = []string{
	"CAST(session_id AS TEXT)",
	"note",
	"group_name",
	"target",
	"type",
	"pipeline_id",
	"listener_id",
	"profile_name",
	"data",
}

// ListSessionsPage returns one stable, offset-based page and the exact filtered total.
func ListSessionsPage(options SessionListOptions) (*SessionListResult, error) {
	return ListSessionsPageContext(context.Background(), options)
}

// ListSessionsPageContext is ListSessionsPage with request cancellation support.
func ListSessionsPageContext(ctx context.Context, options SessionListOptions) (*SessionListResult, error) {
	options, offset, err := normalizeSessionListOptions(options)
	if err != nil {
		return nil, err
	}

	database := Session()
	if database == nil {
		return nil, errors.New("database is not initialized")
	}
	database = database.WithContext(ctx)

	result := &SessionListResult{
		Sessions: make(Sessions, 0, options.PageSize),
		Page:     options.Page,
		PageSize: options.PageSize,
	}

	err = database.Transaction(func(tx *gorm.DB) error {
		stats, err := querySessionStats(tx, SessionStatsOptions{
			Group:  options.Group,
			Search: options.Search,
		})
		if err != nil {
			return err
		}
		result.Stats = stats
		result.Total = sessionListFilteredTotal(stats, options.Status)
		if result.Total == 0 || int64(offset) >= result.Total {
			return nil
		}

		var pageRows []struct {
			SessionID string `gorm:"column:session_id"`
		}
		pageQuery := sessionListBaseQuery(tx, options).Select("session_id")
		pageQuery = applySessionListOrder(pageQuery, options.Sort)
		if err := pageQuery.Offset(offset).Limit(options.PageSize).Scan(&pageRows).Error; err != nil {
			return err
		}

		ids := make([]string, 0, len(pageRows))
		for _, row := range pageRows {
			ids = append(ids, row.SessionID)
		}
		if len(ids) == 0 {
			return nil
		}

		var loaded Sessions
		if err := tx.Where("session_id IN ?", ids).Find(&loaded).Error; err != nil {
			return err
		}

		byID := make(map[string]*models.Session, len(loaded))
		for _, session := range loaded {
			byID[session.SessionID] = session
		}
		for _, id := range ids {
			session, ok := byID[id]
			if !ok {
				return fmt.Errorf("session %q disappeared while loading page", id)
			}
			result.Sessions = append(result.Sessions, session)
		}
		return nil
	}, sessionListTransactionOptions(database))
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetSessionStats returns exact counts for the same group and search filters
// used by ListSessionsPage.
func GetSessionStats(options SessionStatsOptions) (*SessionStats, error) {
	return GetSessionStatsContext(context.Background(), options)
}

// GetSessionStatsContext is GetSessionStats with request cancellation support.
func GetSessionStatsContext(ctx context.Context, options SessionStatsOptions) (*SessionStats, error) {
	database := Session()
	if database == nil {
		return nil, errors.New("database is not initialized")
	}
	database = database.WithContext(ctx)

	stats, err := querySessionStats(database, options)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func querySessionStats(database *gorm.DB, options SessionStatsOptions) (SessionStats, error) {
	var stats SessionStats
	err := sessionListBaseQuery(database, SessionListOptions{
		Group:  options.Group,
		Search: options.Search,
	}).Select(
		"COUNT(*) AS total, COALESCE(SUM(CASE WHEN is_alive = ? THEN 1 ELSE 0 END), 0) AS alive",
		true,
	).Scan(&stats).Error
	if err != nil {
		return SessionStats{}, err
	}
	stats.Offline = stats.Total - stats.Alive
	return stats, nil
}

func sessionListFilteredTotal(stats SessionStats, status SessionListStatus) int64 {
	switch status {
	case SessionListStatusAlive:
		return stats.Alive
	case SessionListStatusOffline:
		return stats.Offline
	default:
		return stats.Total
	}
}

// ListSessionGroups returns distinct non-empty groups used by visible sessions.
func ListSessionGroups() ([]string, error) {
	return ListSessionGroupsContext(context.Background())
}

// ListSessionGroupsContext is ListSessionGroups with request cancellation support.
func ListSessionGroupsContext(ctx context.Context) ([]string, error) {
	database := Session()
	if database == nil {
		return nil, errors.New("database is not initialized")
	}
	database = database.WithContext(ctx)

	groups := make([]string, 0)
	err := database.Model(&models.Session{}).
		Distinct("group_name").
		Where("is_removed = ?", false).
		Where("group_name <> ''").
		Order(clause.OrderByColumn{Column: clause.Column{Name: "group_name"}}).
		Pluck("group_name", &groups).Error
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func normalizeSessionListOptions(options SessionListOptions) (SessionListOptions, int, error) {
	if options.Page == 0 {
		options.Page = DefaultSessionListPage
	}
	if options.PageSize == 0 {
		options.PageSize = DefaultSessionListPageSize
	}
	if options.Page < 1 {
		return options, 0, fmt.Errorf("%w: page must be at least 1", ErrInvalidSessionListOptions)
	}
	if options.PageSize < 1 || options.PageSize > MaxSessionListPageSize {
		return options, 0, fmt.Errorf("%w: page size must be between 1 and %d", ErrInvalidSessionListOptions, MaxSessionListPageSize)
	}
	switch options.Status {
	case "", SessionListStatusAll, SessionListStatusAlive, SessionListStatusOffline:
	default:
		return options, 0, fmt.Errorf("%w: unsupported status %q", ErrInvalidSessionListOptions, options.Status)
	}
	if options.Sort != nil {
		if _, ok := sessionListSortColumns[options.Sort.Field]; !ok {
			return options, 0, fmt.Errorf("%w: unsupported sort field %q", ErrInvalidSessionListOptions, options.Sort.Field)
		}
		switch options.Sort.Direction {
		case SessionListSortAscending, SessionListSortDescending:
		default:
			return options, 0, fmt.Errorf("%w: unsupported sort direction %q", ErrInvalidSessionListOptions, options.Sort.Direction)
		}
	}

	pageIndex := int64(options.Page - 1)
	if pageIndex > int64(math.MaxInt)/int64(options.PageSize) {
		return options, 0, fmt.Errorf("%w: page offset is too large", ErrInvalidSessionListOptions)
	}
	offset64 := pageIndex * int64(options.PageSize)
	return options, int(offset64), nil
}

func sessionListBaseQuery(tx *gorm.DB, options SessionListOptions) *gorm.DB {
	query := tx.Model(&models.Session{}).Where("is_removed = ?", false)
	switch options.Status {
	case SessionListStatusAlive:
		query = query.Where("is_alive = ?", true)
	case SessionListStatusOffline:
		query = query.Where("is_alive = ?", false)
	}
	if options.Group != "" {
		query = query.Where("group_name = ?", options.Group)
	}
	if options.Search != "" {
		predicate := make([]string, 0, len(sessionListSearchColumns))
		args := make([]interface{}, 0, len(sessionListSearchColumns))
		pattern := "%" + escapeSessionListSearch(strings.ToLower(options.Search)) + "%"
		for _, column := range sessionListSearchColumns {
			predicate = append(predicate, sessionListSearchPredicate(tx.Dialector.Name(), column))
			args = append(args, pattern)
		}
		query = query.Where("("+strings.Join(predicate, " OR ")+")", args...)
	}
	return query
}

func sessionListSearchPredicate(dialect, column string) string {
	if dialect == "sqlite" {
		return column + " LIKE ? ESCAPE '!'"
	}
	return "LOWER(COALESCE(" + column + ", '')) LIKE ? ESCAPE '!'"
}

func applySessionListOrder(query *gorm.DB, sort *SessionListSort) *gorm.DB {
	if sort == nil {
		return query.
			Order(clause.OrderByColumn{Column: clause.Column{Name: "is_alive"}, Desc: true}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: true}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "session_id"}})
	}

	if sort.Field == SessionListSortFieldIsAlive {
		descending := sort.Direction == SessionListSortDescending
		return query.
			Order(clause.OrderByColumn{Column: clause.Column{Name: "is_alive"}, Desc: descending}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: descending}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "session_id"}, Desc: !descending})
	}

	column := sessionListSortColumns[sort.Field]
	query = query.Order(clause.OrderByColumn{
		Column: clause.Column{Name: column},
		Desc:   sort.Direction == SessionListSortDescending,
	})
	query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: "is_alive"}, Desc: true})
	if sort.Field != SessionListSortFieldCreatedAt {
		query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: true})
	}
	if sort.Field != SessionListSortFieldSessionID {
		query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: "session_id"}})
	}
	return query
}

func escapeSessionListSearch(value string) string {
	return strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(value)
}

func sessionListTransactionOptions(database *gorm.DB) *sql.TxOptions {
	options := &sql.TxOptions{}
	if database.Dialector.Name() == "postgres" {
		options.ReadOnly = true
		options.Isolation = sql.LevelRepeatableRead
	}
	return options
}
