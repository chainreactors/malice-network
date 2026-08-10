package build

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func TestSaasBuilderExecuteResultStatus(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		closeFirst bool
		buildName  string
		wantStatus string
		wantErr    bool
	}{
		{
			name: "accepted",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantStatus: consts.BuildStatusRunning,
		},
		{
			name:       "network error",
			handler:    func(http.ResponseWriter, *http.Request) {},
			closeFirst: true,
			wantStatus: consts.BuildStatusNetworkError,
			wantErr:    true,
		},
		{
			name: "HTTP failure",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			},
			wantStatus: consts.BuildStatusFailure,
			wantErr:    true,
		},
		{
			name: "marshal failure",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			buildName:  string([]byte{0xff}),
			wantStatus: consts.BuildStatusFailure,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs.InitTestConfigRuntime(t)
			if err := configs.UpdateSaasConfig(&configs.SaasConfig{Enable: true, Token: "test-token"}); err != nil {
				t.Fatalf("configure SaaS: %v", err)
			}
			server := httptest.NewServer(tt.handler)
			if tt.closeFirst {
				server.Close()
			} else {
				defer server.Close()
			}

			buildName := tt.buildName
			if buildName == "" {
				buildName = "execute-test"
			}
			artifact := &models.Artifact{Name: "execute-test"}
			builder := &SaasBuilder{
				config:     &clientpb.BuildConfig{BuildName: buildName},
				builder:    artifact,
				executeUrl: server.URL,
			}
			err := builder.Execute()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if artifact.Status != tt.wantStatus {
				t.Fatalf("artifact status = %q, want %q", artifact.Status, tt.wantStatus)
			}
			if err != nil && StatusFromError(err) != tt.wantStatus {
				t.Fatalf("StatusFromError() = %q, want %q", StatusFromError(err), tt.wantStatus)
			}
		})
	}
}
