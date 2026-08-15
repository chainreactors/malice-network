package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultSessionListBenchmarkRows = 10_000
	maxSessionListBenchmarkRows     = 1_000_000
	sessionListBenchmarkBatchSize   = 500
	sessionListBenchmarkPageSize    = 10
)

var sessionListBenchmarkSink *SessionListResult

type sessionListBenchmarkFixture struct {
	rows         int
	visible      int64
	alive        int64
	group03      int64
	searchNeedle int64
}

// Run at one million rows with:
// MALICE_SESSION_BENCH_ROWS=1000000 go test ./server/internal/db -run '^$' -bench '^BenchmarkListSessionsPage$' -benchmem
func BenchmarkListSessionsPage(b *testing.B) {
	rowCount := sessionListBenchmarkRows(b)
	fixture := prepareSessionListBenchmark(b, rowCount)

	deepPage := sessionListBenchmarkDeepPage(fixture.visible, sessionListBenchmarkPageSize)
	cases := []struct {
		name          string
		options       SessionListOptions
		expectedTotal int64
	}{
		{
			name: "DefaultPage",
			options: SessionListOptions{
				Page:     1,
				PageSize: sessionListBenchmarkPageSize,
			},
			expectedTotal: fixture.visible,
		},
		{
			name: "DeepPage",
			options: SessionListOptions{
				Page:     deepPage,
				PageSize: sessionListBenchmarkPageSize,
			},
			expectedTotal: fixture.visible,
		},
		{
			name: "AliveFilter",
			options: SessionListOptions{
				Page:     1,
				PageSize: sessionListBenchmarkPageSize,
				Status:   SessionListStatusAlive,
			},
			expectedTotal: fixture.alive,
		},
		{
			name: "GroupFilter",
			options: SessionListOptions{
				Page:     1,
				PageSize: sessionListBenchmarkPageSize,
				Group:    "group-03",
			},
			expectedTotal: fixture.group03,
		},
		{
			name: "FuzzySearchWithExactCount",
			options: SessionListOptions{
				Page:     1,
				PageSize: sessionListBenchmarkPageSize,
				Search:   "needle",
			},
			expectedTotal: fixture.searchNeedle,
		},
	}

	for _, benchmarkCase := range cases {
		b.Run(benchmarkCase.name, func(b *testing.B) {
			b.StopTimer()
			warm, err := ListSessionsPage(benchmarkCase.options)
			if err != nil {
				b.Fatalf("warm query failed: %v", err)
			}
			if warm.Total != benchmarkCase.expectedTotal {
				b.Fatalf("filtered total = %d, want %d", warm.Total, benchmarkCase.expectedTotal)
			}
			rowsPerQuery := len(warm.Sessions)
			b.ReportAllocs()

			b.ResetTimer()
			b.StartTimer()
			startedAt := time.Now()
			for i := 0; i < b.N; i++ {
				result, err := ListSessionsPage(benchmarkCase.options)
				if err != nil {
					b.Fatalf("ListSessionsPage failed: %v", err)
				}
				if result.Total != benchmarkCase.expectedTotal {
					b.Fatalf("filtered total = %d, want %d", result.Total, benchmarkCase.expectedTotal)
				}
				sessionListBenchmarkSink = result
			}
			elapsed := time.Since(startedAt)
			b.StopTimer()

			b.ReportMetric(float64(fixture.rows), "dataset_rows")
			if elapsed > 0 {
				b.ReportMetric(float64(b.N*rowsPerQuery)/elapsed.Seconds(), "rows/s")
			}
		})
	}
}

func sessionListBenchmarkRows(b *testing.B) int {
	b.Helper()

	value := os.Getenv("MALICE_SESSION_BENCH_ROWS")
	if value == "" {
		return defaultSessionListBenchmarkRows
	}
	rows, err := strconv.Atoi(value)
	if err != nil || rows < 1_000 || rows > maxSessionListBenchmarkRows {
		b.Fatalf(
			"MALICE_SESSION_BENCH_ROWS must be an integer between 1000 and %d, got %q",
			maxSessionListBenchmarkRows,
			value,
		)
	}
	return rows
}

func prepareSessionListBenchmark(b *testing.B, rowCount int) sessionListBenchmarkFixture {
	b.Helper()

	dsn := "file:" + filepath.ToSlash(filepath.Join(b.TempDir(), "sessions.db")) +
		"?_pragma=journal_mode(WAL)&_pragma=synchronous(off)&_pragma=temp_store(memory)&_pragma=cache_size(-262144)"
	database, err := gorm.Open(Open(dsn), &gorm.Config{
		PrepareStmt: false,
		Logger:      logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		b.Fatalf("open benchmark database: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		b.Fatalf("get benchmark sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	previousClient := Client
	previousAdapter := Adapter
	Client = database
	Adapter = &sqliteAdapter{}
	b.Cleanup(func() {
		Client = previousClient
		Adapter = previousAdapter
		if err := sqlDB.Close(); err != nil {
			b.Errorf("close benchmark database: %v", err)
		}
	})

	if err := database.AutoMigrate(&models.Session{}); err != nil {
		b.Fatalf("migrate benchmark database: %v", err)
	}
	indexes := []string{
		"idx_sessions_profile_name",
		"idx_sessions_list_default",
		"idx_sessions_list_group",
		"idx_sessions_trend",
	}
	for _, index := range indexes {
		if database.Migrator().HasIndex(&models.Session{}, index) {
			if err := database.Exec("DROP INDEX IF EXISTS " + index).Error; err != nil {
				b.Fatalf("drop fixture index %q: %v", index, err)
			}
		}
	}

	fixture := insertSessionListBenchmarkRows(b, sqlDB, rowCount)

	for _, index := range indexes {
		if err := database.Migrator().CreateIndex(&models.Session{}, index); err != nil {
			b.Fatalf("create fixture index %q: %v", index, err)
		}
	}
	if err := database.Exec("ANALYZE sessions").Error; err != nil {
		b.Fatalf("analyze benchmark sessions: %v", err)
	}
	if err := database.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error; err != nil {
		b.Fatalf("checkpoint benchmark database: %v", err)
	}
	return fixture
}

func insertSessionListBenchmarkRows(b *testing.B, sqlDB *sql.DB, rowCount int) sessionListBenchmarkFixture {
	b.Helper()

	tx, err := sqlDB.BeginTx(context.Background(), nil)
	if err != nil {
		b.Fatalf("begin fixture transaction: %v", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	groups := make([]string, 16)
	pipelines := make([]string, 8)
	listeners := make([]string, 8)
	for i := range groups {
		groups[i] = fmt.Sprintf("group-%02d", i)
	}
	for i := range pipelines {
		pipelines[i] = fmt.Sprintf("pipeline-%02d", i)
		listeners[i] = fmt.Sprintf("listener-%02d", i)
	}

	const (
		normalData = "{\"locale\":\"en-US\",\"filepath\":\"/opt/agent\",\"modules\":[\"execute\",\"sys\"],\"addons\":[],\"argue\":{}}"
		needleData = "{\"locale\":\"en-US\",\"filepath\":\"/opt/agent\",\"modules\":[\"execute\",\"sys\"],\"addons\":[],\"argue\":{},\"bench_marker\":\"needle\"}"
	)
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fixture := sessionListBenchmarkFixture{rows: rowCount}

	for start := 0; start < rowCount; start += sessionListBenchmarkBatchSize {
		end := start + sessionListBenchmarkBatchSize
		if end > rowCount {
			end = rowCount
		}
		args := make([]interface{}, 0, (end-start)*15)
		for i := start; i < end; i++ {
			group := groups[i%len(groups)]
			removed := i%100 == 0
			alive := i%20 == 1
			data := normalData
			if i%1000 == 7 {
				data = needleData
			}
			if !removed {
				fixture.visible++
				if alive {
					fixture.alive++
				}
				if group == "group-03" {
					fixture.group03++
				}
				if data == needleData {
					fixture.searchNeedle++
				}
			}

			sessionType := "beacon"
			if i%10 == 0 {
				sessionType = "bind"
			}
			target := "linux/amd64"
			if i%3 == 0 {
				target = "windows/amd64"
			}
			args = append(args,
				fmt.Sprintf("%08x-0000-4000-8000-%012x", i, i),
				uint32(i+1),
				baseTime.Add(time.Duration(i)*time.Second),
				"benchmark session",
				group,
				target,
				true,
				sessionType,
				pipelines[i%len(pipelines)],
				listeners[i%len(listeners)],
				alive,
				baseTime.Unix()+int64(i),
				removed,
				data,
				"profile-benchmark",
			)
		}
		if _, err := tx.Exec(sessionListBenchmarkInsertSQL(end-start), args...); err != nil {
			b.Fatalf("insert benchmark rows %d-%d: %v", start, end, err)
		}
	}
	if err := tx.Commit(); err != nil {
		b.Fatalf("commit fixture transaction: %v", err)
	}
	tx = nil
	return fixture
}

func sessionListBenchmarkInsertSQL(rowCount int) string {
	const rowPlaceholders = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	var query strings.Builder
	query.Grow(256 + rowCount*(len(rowPlaceholders)+2))
	query.WriteString(
		"INSERT INTO sessions (" +
			"session_id, raw_id, created_at, note, group_name, target, initialized, type, " +
			"pipeline_id, listener_id, is_alive, last_checkin, is_removed, data, profile_name" +
			") VALUES ",
	)
	for i := 0; i < rowCount; i++ {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(rowPlaceholders)
	}
	return query.String()
}

func sessionListBenchmarkDeepPage(total int64, pageSize int) int {
	if total <= int64(pageSize) {
		return 1
	}
	maxOffset := total - int64(pageSize)
	targetOffset := total * 9 / 10
	if targetOffset > maxOffset {
		targetOffset = maxOffset
	}
	return int(targetOffset/int64(pageSize)) + 1
}
