package db

import (
	"testing"
	"time"

	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func TestGetSessionTrend_ReturnsFixedBucketsAndEnforcesWindowBounds(t *testing.T) {
	initTestDB(t)

	now := time.Date(2026, 8, 15, 12, 34, 56, 0, time.UTC)
	lastBucketStart := now.Truncate(time.Hour)
	firstBucketStart := lastBucketStart.Add(-(SessionTrendBucketCount - 1) * time.Hour)

	fixtures := []struct {
		id        string
		createdAt time.Time
		removed   bool
	}{
		{id: "before-window", createdAt: firstBucketStart.Add(-time.Second)},
		{id: "at-window-start", createdAt: firstBucketStart},
		{id: "inside-oldest-hour", createdAt: firstBucketStart.Add(59 * time.Minute)},
		{id: "middle-hour", createdAt: lastBucketStart.Add(-2*time.Hour + 15*time.Minute)},
		{id: "current-hour", createdAt: lastBucketStart.Add(5 * time.Minute)},
		// The current hour is the final bucket, so its full half-open hour is counted.
		{id: "later-in-current-hour", createdAt: lastBucketStart.Add(50 * time.Minute)},
		{id: "at-next-hour", createdAt: lastBucketStart.Add(time.Hour)},
		{id: "far-future", createdAt: lastBucketStart.Add(24 * time.Hour)},
		{id: "removed-current", createdAt: lastBucketStart.Add(10 * time.Minute), removed: true},
	}
	for _, fixture := range fixtures {
		createSessionListFixture(t, &models.Session{
			SessionID: fixture.id,
			IsRemoved: fixture.removed,
		}, fixture.createdAt)
	}

	points, err := GetSessionTrend(now)
	if err != nil {
		t.Fatalf("GetSessionTrend failed: %v", err)
	}
	if len(points) != SessionTrendBucketCount {
		t.Fatalf("point count = %d, want %d", len(points), SessionTrendBucketCount)
	}

	wantCounts := map[int64]int64{
		firstBucketStart.Unix():                    2,
		lastBucketStart.Add(-2 * time.Hour).Unix(): 1,
		lastBucketStart.Unix():                     2,
	}
	for i, point := range points {
		wantStart := firstBucketStart.Add(time.Duration(i) * time.Hour).Unix()
		if point.BucketStartUnix != wantStart {
			t.Fatalf("point %d start = %d, want %d", i, point.BucketStartUnix, wantStart)
		}
		if point.Count != wantCounts[wantStart] {
			t.Fatalf("point %d count = %d, want %d", i, point.Count, wantCounts[wantStart])
		}
	}
}

func TestSessionTrendBucketExpression(t *testing.T) {
	tests := []struct {
		dialect string
		wantErr bool
	}{
		{dialect: "sqlite"},
		{dialect: "postgres"},
		{dialect: "mysql", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.dialect, func(t *testing.T) {
			expression, err := sessionTrendBucketExpression(test.dialect)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && expression == "" {
				t.Fatal("bucket expression is empty")
			}
		})
	}
}
