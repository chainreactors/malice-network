package rpc

import "testing"

func TestMatchMethodSupportsExactAndWildcards(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		method  string
		want    bool
	}{
		{
			name:    "exact",
			pattern: "/clientrpc.MaliceRPC/GetSessions",
			method:  "/clientrpc.MaliceRPC/GetSessions",
			want:    true,
		},
		{
			name:    "service wildcard",
			pattern: "/clientrpc.MaliceRPC/*",
			method:  "/clientrpc.MaliceRPC/GetSessions",
			want:    true,
		},
		{
			name:    "package wildcard",
			pattern: "/listenerrpc.*",
			method:  "/listenerrpc.ListenerRPC/SpiteStream",
			want:    true,
		},
		{
			name:    "different service",
			pattern: "/clientrpc.RootRPC/*",
			method:  "/clientrpc.MaliceRPC/GetSessions",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchMethod(tt.pattern, tt.method); got != tt.want {
				t.Fatalf("matchMethod(%q, %q) = %v, want %v", tt.pattern, tt.method, got, tt.want)
			}
		})
	}
}
