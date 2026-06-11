package agent

import (
	"testing"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestChatBackendForSession(t *testing.T) {
	tests := []struct {
		name string
		sess *client.Session
		want chatBackend
	}{
		{
			name: "nil session uses dedicated backend",
			sess: nil,
			want: chatBackendDedicated,
		},
		{
			name: "bridge target uses bridge backend",
			sess: &client.Session{Session: &clientpb.Session{Target: "llm-agent://claude"}},
			want: chatBackendBridge,
		},
		{
			name: "bridge target match is case insensitive",
			sess: &client.Session{Session: &clientpb.Session{Target: "LLM-Agent://openai"}},
			want: chatBackendBridge,
		},
		{
			name: "normal implant session uses dedicated backend",
			sess: &client.Session{Session: &clientpb.Session{Target: "tcp://10.0.0.5"}},
			want: chatBackendDedicated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatBackendForSession(tt.sess); got != tt.want {
				t.Fatalf("chatBackendForSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultChatOptionsBridgeSkipsAISettings(t *testing.T) {
	sess := &client.Session{Session: &clientpb.Session{Target: "llm-agent://bridge"}}

	opts, err := defaultChatOptions(sess, "list files")
	if err != nil {
		t.Fatalf("defaultChatOptions() returned error for bridge session: %v", err)
	}
	if opts.Text != "list files" {
		t.Fatalf("defaultChatOptions().Text = %q, want %q", opts.Text, "list files")
	}
	if opts.Model != "" {
		t.Fatalf("defaultChatOptions().Model = %q, want empty string", opts.Model)
	}
	if opts.Provider != "" {
		t.Fatalf("defaultChatOptions().Provider = %q, want empty string", opts.Provider)
	}
}
