package db

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func TestTailBuilderLog(t *testing.T) {
	tests := []struct {
		name  string
		log   string
		limit int
		want  string
	}{
		{
			name:  "trailing newline does not consume the line limit",
			log:   "line1\nline2\nline3\nline4\n",
			limit: 2,
			want:  "line3\nline4\n",
		},
		{
			name:  "log without trailing newline",
			log:   "line1\nline2\nline3",
			limit: 2,
			want:  "line2\nline3",
		},
		{
			name:  "CRLF line endings",
			log:   "line1\r\nline2\r\nline3\r\n",
			limit: 2,
			want:  "line2\r\nline3\r\n",
		},
		{
			name:  "blank lines count as log lines",
			log:   "line1\n\nline3\n",
			limit: 2,
			want:  "\nline3\n",
		},
		{
			name:  "zero returns the complete log",
			log:   "line1\nline2\n",
			limit: 0,
			want:  "line1\nline2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tailBuilderLog(tt.log, tt.limit); got != tt.want {
				t.Fatalf("tailBuilderLog() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetBuilderLogsSelectsOnlyLog(t *testing.T) {
	initTestDB(t)

	artifact := &models.Artifact{
		Name:       "log-only-query",
		Log:        "line1\nline2\nline3\n",
		ParamsData: "{invalid-json",
	}
	if err := Session().Create(artifact).Error; err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	got, err := GetBuilderLogs(artifact.Name, 2)
	if err != nil {
		t.Fatalf("GetBuilderLogs should not deserialize unrelated artifact fields: %v", err)
	}
	if want := "line2\nline3\n"; got != want {
		t.Fatalf("GetBuilderLogs() = %q, want %q", got, want)
	}
}

func TestUpdateBuilderResultUpdatesStatusAndAppendsLog(t *testing.T) {
	initTestDB(t)

	artifact := &models.Artifact{
		Name:   "terminal-result",
		Status: consts.BuildStatusRunning,
		Log:    "existing\n",
	}
	if err := Session().Create(artifact).Error; err != nil {
		t.Fatalf("create artifact: %v", err)
	}

	if err := UpdateBuilderResult(artifact.ID, consts.BuildStatusNetworkError, "network failure\n"); err != nil {
		t.Fatalf("UpdateBuilderResult failed: %v", err)
	}

	var got models.Artifact
	if err := Session().Select("status", "log").First(&got, artifact.ID).Error; err != nil {
		t.Fatalf("load artifact result: %v", err)
	}
	if got.Status != consts.BuildStatusNetworkError {
		t.Fatalf("status = %q, want %q", got.Status, consts.BuildStatusNetworkError)
	}
	if got.Log != "existing\nnetwork failure\n" {
		t.Fatalf("log = %q, want appended terminal log", got.Log)
	}
}
