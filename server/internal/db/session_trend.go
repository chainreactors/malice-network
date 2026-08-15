package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/gorm/clause"
)

const SessionTrendBucketCount = 24

type SessionTrendPoint struct {
	BucketStartUnix int64
	Count           int64
}

// GetSessionTrend returns the 24 UTC hour buckets ending with the hour that
// contains now. The query window is [oldest bucket, next hour), so timestamps
// in older hours and timestamps at or after the next hour are excluded.
func GetSessionTrend(now time.Time) ([]SessionTrendPoint, error) {
	return GetSessionTrendContext(context.Background(), now)
}

// GetSessionTrendContext is GetSessionTrend with request cancellation support.
func GetSessionTrendContext(ctx context.Context, now time.Time) ([]SessionTrendPoint, error) {
	database := Session()
	if database == nil {
		return nil, errors.New("database is not initialized")
	}
	database = database.WithContext(ctx)

	bucketExpression, err := sessionTrendBucketExpression(database.Dialector.Name())
	if err != nil {
		return nil, err
	}

	lastBucketStart := now.UTC().Truncate(time.Hour)
	firstBucketStart := lastBucketStart.Add(-(SessionTrendBucketCount - 1) * time.Hour)
	windowEnd := lastBucketStart.Add(time.Hour)

	type bucketRow struct {
		BucketStartUnix int64 `gorm:"column:bucket_start_unix"`
		Count           int64 `gorm:"column:count"`
	}
	var rows []bucketRow
	if err := database.Model(&models.Session{}).
		Select(bucketExpression+" AS bucket_start_unix, COUNT(*) AS count").
		Where("is_removed = ?", false).
		Where("created_at >= ? AND created_at < ?", firstBucketStart, windowEnd).
		Group(bucketExpression).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "bucket_start_unix"}}).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := make(map[int64]int64, len(rows))
	for _, row := range rows {
		counts[row.BucketStartUnix] = row.Count
	}

	points := make([]SessionTrendPoint, SessionTrendBucketCount)
	for i := range points {
		bucketStart := firstBucketStart.Add(time.Duration(i) * time.Hour).Unix()
		points[i] = SessionTrendPoint{
			BucketStartUnix: bucketStart,
			Count:           counts[bucketStart],
		}
	}
	return points, nil
}

func sessionTrendBucketExpression(dialect string) (string, error) {
	switch dialect {
	case "sqlite":
		return "(CAST(strftime('%s', created_at) AS INTEGER) / 3600) * 3600", nil
	case "postgres":
		return "FLOOR(EXTRACT(EPOCH FROM created_at) / 3600)::bigint * 3600", nil
	default:
		return "", fmt.Errorf("unsupported database dialect %q", dialect)
	}
}
