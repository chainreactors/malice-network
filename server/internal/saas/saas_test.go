package saas

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
