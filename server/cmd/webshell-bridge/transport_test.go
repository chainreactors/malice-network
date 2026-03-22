package main

import (
	"testing"
)

func TestNewTransportValidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"suo5 HTTP", "suo5://target.com/suo5.jsp"},
		{"suo5 HTTPS", "suo5s://target.com/suo5.jsp"},
		{"suo5 with port", "suo5://target.com:8080/suo5.jsp"},
		{"suo5 with path", "suo5://10.0.0.1/app/suo5.aspx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, err := NewTransport(tt.url)
			if err != nil {
				t.Fatalf("NewTransport(%q) error: %v", tt.url, err)
			}
			if tr.rawURL == nil {
				t.Fatal("rawURL is nil")
			}
			if tr.rawURL.Scheme == "" {
				t.Fatal("rawURL scheme is empty")
			}
			if tr.rawURL.Host == "" {
				t.Fatal("rawURL host is empty")
			}
			if tr.client != nil {
				t.Fatal("client should be initialized lazily")
			}
		})
	}
}

func TestNewTransportInvalidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"empty", ""},
		{"no scheme", "target.com/suo5.jsp"},
		{"bad scheme", "://target.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTransport(tt.url)
			if err == nil {
				t.Fatalf("NewTransport(%q) expected error, got nil", tt.url)
			}
		})
	}
}
