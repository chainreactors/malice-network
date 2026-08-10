package saas

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func TestLicenseRequestsHonorContextDeadline(t *testing.T) {
	tests := []struct {
		name string
		call func(context.Context, *SaasClient) error
	}{
		{
			name: "get license info",
			call: func(ctx context.Context, client *SaasClient) error {
				_, _, err := client.GetLicenseInfoContext(ctx)
				return err
			},
		},
		{
			name: "register license",
			call: func(ctx context.Context, client *SaasClient) error {
				_, err := client.RegisterLicenseContext(ctx)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				<-release
			}))
			defer server.Close()

			client := &SaasClient{BaseURL: server.URL, Token: "test-token"}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()

			err := tt.call(ctx, client)
			close(release)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("license request error = %v, want context deadline exceeded", err)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "unexpected EOF", err: io.ErrUnexpectedEOF, want: true},
		{name: "canceled", err: context.Canceled, want: false},
		{name: "HTTP response error", err: errors.New("unexpected status 500"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNetworkError(tt.err); got != tt.want {
				t.Fatalf("IsNetworkError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestCheckAndDownloadArtifactResults(t *testing.T) {
	t.Run("completed", func(t *testing.T) {
		client, builder, closeServer := newBuildTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/status":
				_, _ = io.WriteString(w, `{"success":true,"status":"completed"}`)
			case "/download":
				_, _ = io.WriteString(w, "artifact-content")
			default:
				http.NotFound(w, r)
			}
		})
		defer closeServer()

		result := client.CheckAndDownloadArtifact("/status", "/download", builder, time.Millisecond, 100*time.Millisecond)
		if result.Status != consts.BuildStatusCompleted || result.Err != nil {
			t.Fatalf("result = %#v, want completed", result)
		}
		content, err := os.ReadFile(result.Path)
		if err != nil {
			t.Fatalf("read downloaded artifact: %v", err)
		}
		if string(content) != "artifact-content" {
			t.Fatalf("artifact content = %q", content)
		}
	})

	t.Run("remote failure", func(t *testing.T) {
		client, builder, closeServer := newBuildTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"success":false,"status":"failure"}`)
		})
		defer closeServer()

		result := client.CheckAndDownloadArtifact("/status", "/download", builder, time.Millisecond, 100*time.Millisecond)
		if result.Status != consts.BuildStatusFailure || result.Stage != BuildStageStatus {
			t.Fatalf("result = %#v, want status failure", result)
		}
		if result.Err == nil || !strings.Contains(result.Err.Error(), "reported build failure") {
			t.Fatalf("error = %v, want remote build failure reason", result.Err)
		}
	})

	t.Run("persistent network error", func(t *testing.T) {
		client, builder, closeServer := newBuildTestClient(t, func(http.ResponseWriter, *http.Request) {})
		closeServer()

		result := client.CheckAndDownloadArtifact("/status", "/download", builder, time.Millisecond, 10*time.Millisecond)
		if result.Status != consts.BuildStatusNetworkError || result.Stage != BuildStageStatus {
			t.Fatalf("result = %#v, want status network error", result)
		}
		if result.Err == nil || !strings.Contains(result.Err.Error(), "last error") {
			t.Fatalf("error = %v, want last network error", result.Err)
		}
	})

	t.Run("healthy polling timeout", func(t *testing.T) {
		client, builder, closeServer := newBuildTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"success":true,"status":"running"}`)
		})
		defer closeServer()

		result := client.CheckAndDownloadArtifact("/status", "/download", builder, time.Millisecond, 10*time.Millisecond)
		if result.Status != consts.BuildStatusFailure || result.Stage != BuildStageStatus {
			t.Fatalf("result = %#v, want build timeout failure", result)
		}
		if result.Err == nil || !strings.Contains(result.Err.Error(), "did not complete") {
			t.Fatalf("error = %v, want build timeout reason", result.Err)
		}
	})

	t.Run("transient network error recovers", func(t *testing.T) {
		var statusRequests atomic.Int32
		client, builder, closeServer := newBuildTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/status":
				if statusRequests.Add(1) == 1 {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						_ = conn.Close()
						return
					}
				}
				_, _ = io.WriteString(w, `{"success":true,"status":"completed"}`)
			case "/download":
				_, _ = io.WriteString(w, "artifact-content")
			}
		})
		defer closeServer()

		result := client.CheckAndDownloadArtifact("/status", "/download", builder, time.Millisecond, 100*time.Millisecond)
		if result.Status != consts.BuildStatusCompleted || result.Err != nil {
			t.Fatalf("result = %#v, want recovery and completion", result)
		}
	})

	t.Run("recovered poll is not classified as network timeout", func(t *testing.T) {
		var statusRequests atomic.Int32
		client, builder, closeServer := newBuildTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			if statusRequests.Add(1) == 1 {
				conn, _, err := w.(http.Hijacker).Hijack()
				if err == nil {
					_ = conn.Close()
					return
				}
			}
			_, _ = io.WriteString(w, `{"success":true,"status":"running"}`)
		})
		defer closeServer()

		result := client.CheckAndDownloadArtifact("/status", "/download", builder, time.Millisecond, 10*time.Millisecond)
		if result.Status != consts.BuildStatusFailure || result.Stage != BuildStageStatus {
			t.Fatalf("result = %#v, want ordinary build timeout", result)
		}
	})

	t.Run("HTTP status error is not a network error", func(t *testing.T) {
		client, builder, closeServer := newBuildTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		})
		defer closeServer()

		result := client.CheckAndDownloadArtifact("/status", "/download", builder, time.Millisecond, 100*time.Millisecond)
		if result.Status != consts.BuildStatusFailure || result.Stage != BuildStageStatus {
			t.Fatalf("result = %#v, want ordinary failure", result)
		}
	})

	t.Run("download network error", func(t *testing.T) {
		client, builder, closeServer := newBuildTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/status" {
				_, _ = io.WriteString(w, `{"success":true,"status":"completed"}`)
				return
			}
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.Close()
			}
		})
		defer closeServer()

		result := client.CheckAndDownloadArtifact("/status", "/download", builder, time.Millisecond, 100*time.Millisecond)
		if result.Status != consts.BuildStatusNetworkError || result.Stage != BuildStageDownload {
			t.Fatalf("result = %#v, want download network error", result)
		}
	})
}

func TestFormatBuildResultLog(t *testing.T) {
	now := time.Date(2026, time.August, 10, 10, 3, 21, 0, time.FixedZone("CST", 8*60*60))
	tests := []struct {
		name   string
		result BuildResult
		want   string
	}{
		{
			name:   "completed",
			result: BuildResult{Status: consts.BuildStatusCompleted},
			want:   "2026-08-10T10:03:21+08:00 [COMPLETED] SaaS build completed\n",
		},
		{
			name:   "failure",
			result: BuildResult{Status: consts.BuildStatusFailure, Stage: BuildStageStatus, Err: errors.New("remote\nfailed")},
			want:   "2026-08-10T10:03:21+08:00 [FAILED] stage=status reason=remote failed\n",
		},
		{
			name:   "network error",
			result: BuildResult{Status: consts.BuildStatusNetworkError, Stage: BuildStageDownload, Err: errors.New("connection reset")},
			want:   "2026-08-10T10:03:21+08:00 [NETWORK_ERROR] stage=download reason=connection reset\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBuildResultLog(now, tt.result); got != tt.want {
				t.Fatalf("formatBuildResultLog() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildResultReasonIsSingleLineAndBounded(t *testing.T) {
	reason := buildResultReason(errors.New(strings.Repeat("x", maxBuildResultReasonRunes+100) + "\nsecret"))
	if strings.ContainsAny(reason, "\r\n") {
		t.Fatalf("reason contains a newline: %q", reason)
	}
	if got := len([]rune(reason)); got != maxBuildResultReasonRunes {
		t.Fatalf("reason length = %d, want %d", got, maxBuildResultReasonRunes)
	}
}

func newBuildTestClient(t *testing.T, handler http.HandlerFunc) (*SaasClient, *models.Artifact, func()) {
	t.Helper()
	configs.UseTestPaths(t, t.TempDir())
	if err := os.MkdirAll(configs.TempPath, 0700); err != nil {
		t.Fatalf("create temp path: %v", err)
	}
	server := httptest.NewServer(handler)
	return &SaasClient{BaseURL: server.URL, Token: "test-token"}, &models.Artifact{Name: "test-artifact"}, server.Close
}
