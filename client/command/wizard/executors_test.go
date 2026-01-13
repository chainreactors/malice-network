package wizard

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
)

func TestParseAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantAddr string
		wantTCP  bool
		wantHTTP bool
		wantTLS  bool
		wantSNI  string
		wantErr  bool
	}{
		{
			name:     "http default port",
			input:    "http://example.com",
			wantAddr: "example.com:80",
			wantHTTP: true,
		},
		{
			name:     "https default port and sni",
			input:    "https://example.com",
			wantAddr: "example.com:443",
			wantHTTP: true,
			wantTLS:  true,
			wantSNI:  "example.com",
		},
		{
			name:     "tcp default port",
			input:    "tcp://example.com",
			wantAddr: "example.com:5001",
			wantTCP:  true,
		},
		{
			name:     "tcp+tls default port and sni",
			input:    "tcp+tls://example.com",
			wantAddr: "example.com:5001",
			wantTCP:  true,
			wantTLS:  true,
			wantSNI:  "example.com",
		},
		{
			name:     "raw host defaults to tcp",
			input:    "example.com",
			wantAddr: "example.com:5001",
			wantTCP:  true,
		},
		{
			name:     "raw ipv6 defaults to tcp",
			input:    "::1",
			wantAddr: "[::1]:5001",
			wantTCP:  true,
		},
		{
			name:     "raw ipv6 with port",
			input:    "[::1]:6000",
			wantAddr: "[::1]:6000",
			wantTCP:  true,
		},
		{
			name:    "unsupported scheme",
			input:   "ftp://example.com",
			wantErr: true,
		},
		{
			name:    "http path not allowed",
			input:   "http://example.com/foo",
			wantErr: true,
		},
		{
			name:    "raw address with path not allowed",
			input:   "example.com/foo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			target, err := parseAddress(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (target=%+v)", target)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if target == nil {
				t.Fatalf("expected target, got nil")
			}
			if target.Address != tt.wantAddr {
				t.Fatalf("address mismatch: got %q want %q", target.Address, tt.wantAddr)
			}
			if (target.TCP != nil) != tt.wantTCP {
				t.Fatalf("tcp mismatch: got %v want %v", target.TCP != nil, tt.wantTCP)
			}
			if (target.Http != nil) != tt.wantHTTP {
				t.Fatalf("http mismatch: got %v want %v", target.Http != nil, tt.wantHTTP)
			}
			if (target.TLS != nil) != tt.wantTLS {
				t.Fatalf("tls mismatch: got %v want %v", target.TLS != nil, tt.wantTLS)
			}
			if tt.wantTLS && target.TLS != nil && target.TLS.SNI != tt.wantSNI {
				t.Fatalf("sni mismatch: got %q want %q", target.TLS.SNI, tt.wantSNI)
			}
		})
	}
}

func TestParseTargets_TrimsAndValidates(t *testing.T) {
	t.Parallel()

	targets, err := parseTargets(" http://example.com , tcp://127.0.0.1:5001 , ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("unexpected target count: got %d want %d", len(targets), 2)
	}
	if targets[0].Address != "example.com:80" {
		t.Fatalf("unexpected first address: %q", targets[0].Address)
	}
	if targets[1].Address != "127.0.0.1:5001" {
		t.Fatalf("unexpected second address: %q", targets[1].Address)
	}
}

func TestParsePulseAddress(t *testing.T) {
	t.Parallel()

	parsed, err := parsePulseAddress("tcp://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.protocol != consts.TCPPipeline {
		t.Fatalf("unexpected protocol: got %q want %q", parsed.protocol, consts.TCPPipeline)
	}
	if parsed.target != "example.com:5001" {
		t.Fatalf("unexpected target: got %q", parsed.target)
	}

	parsed, err = parsePulseAddress("http://[::1]/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.protocol != consts.HTTPPipeline {
		t.Fatalf("unexpected protocol: got %q want %q", parsed.protocol, consts.HTTPPipeline)
	}
	if parsed.target != "[::1]:80" {
		t.Fatalf("unexpected target: got %q", parsed.target)
	}

	if _, err := parsePulseAddress("https://example.com"); err == nil {
		t.Fatalf("expected error for https pulse address, got nil")
	}
}
