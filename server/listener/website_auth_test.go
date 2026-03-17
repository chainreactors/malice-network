package listener

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebsiteCheckAuthRejectsUnauthorizedHeadRequest(t *testing.T) {
	web := &Website{defaultAuth: "admin:pass123"}
	req := httptest.NewRequest(http.MethodHead, "http://example.com/secret.html", nil)
	resp := httptest.NewRecorder()

	allowed := web.checkAuth("", resp, req)
	if allowed {
		t.Fatal("checkAuth allowed request without credentials, want rejection")
	}
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status code = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
	if got := resp.Header().Get("WWW-Authenticate"); got == "" {
		t.Fatal("WWW-Authenticate header = empty, want auth challenge")
	}
}

func TestWebsiteCheckAuthAllowsNoneOverride(t *testing.T) {
	web := &Website{defaultAuth: "admin:pass123"}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/public.html", nil)
	resp := httptest.NewRecorder()

	allowed := web.checkAuth("none", resp, req)
	if !allowed {
		t.Fatal("checkAuth denied explicit none override, want allowed")
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", resp.Code, http.StatusOK)
	}
}
