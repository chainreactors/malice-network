package httputils

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoPOSTDoesNotMutateCallerHeaders(t *testing.T) {
	headers := map[string]string{"X-Test": "keep"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		if got := r.Header.Get("X-Test"); got != "keep" {
			t.Fatalf("X-Test = %q, want keep", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"name":"demo"}` {
			t.Fatalf("body = %q, want JSON payload", string(body))
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := DoPOST(server.URL, map[string]string{"name": "demo"}, headers, http.StatusCreated, nil); err != nil {
		t.Fatalf("DoPOST failed: %v", err)
	}
	if _, ok := headers["Content-Type"]; ok {
		t.Fatalf("headers mutated = %#v, want original map unchanged", headers)
	}
}

func TestDoJSONRequestAllowsEmptyResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var out map[string]any
	if err := DoJSONRequest(http.MethodGet, server.URL, nil, nil, http.StatusOK, &out); err != nil {
		t.Fatalf("DoJSONRequest failed on empty body: %v", err)
	}
	if out != nil {
		t.Fatalf("out = %#v, want nil on empty response body", out)
	}
}

func TestDoGETDecodesJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	var out map[string]string
	if err := DoGET(server.URL, nil, &out); err != nil {
		t.Fatalf("DoGET failed: %v", err)
	}
	if out["status"] != "ok" {
		t.Fatalf("out = %#v, want status=ok", out)
	}
}
