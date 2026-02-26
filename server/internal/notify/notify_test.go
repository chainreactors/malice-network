package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLarkWebhookService_SendWithoutSecret(t *testing.T) {
	var body map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"StatusCode":0,"StatusMessage":"success"}`))
	}))
	defer srv.Close()

	svc := NewLarkWebhookService(srv.URL, "")
	err := svc.Send(context.Background(), "test title", "test message")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Without secret, sign should be empty and timestamp should be 0
	if sign, ok := body["sign"]; ok && sign != "" {
		t.Errorf("expected empty sign without secret, got %q", sign)
	}
}

func TestLarkWebhookService_SendWithSecret(t *testing.T) {
	var body map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"StatusCode":0,"StatusMessage":"success"}`))
	}))
	defer srv.Close()

	svc := NewLarkWebhookService(srv.URL, "test-secret-key")
	err := svc.Send(context.Background(), "test title", "test message")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	sign, ok := body["sign"]
	if !ok || sign == "" {
		t.Fatal("expected non-empty sign with secret")
	}

	ts, ok := body["timestamp"]
	if !ok {
		t.Fatal("expected timestamp with secret")
	}
	// go-lark serializes timestamp as string via strconv.FormatInt
	tsStr, isStr := ts.(string)
	if isStr {
		if tsStr == "" || tsStr == "0" {
			t.Error("expected non-zero timestamp")
		}
	} else if tsFloat, isFloat := ts.(float64); isFloat {
		if tsFloat == 0 {
			t.Error("expected non-zero timestamp")
		}
	} else {
		t.Errorf("unexpected timestamp type: %T", ts)
	}
}
