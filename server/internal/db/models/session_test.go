package models

import (
	"testing"

	"github.com/chainreactors/IoM-go/client"
)

func TestSessionToProtobufHandlesMissingContext(t *testing.T) {
	tests := []struct {
		name string
		data *client.SessionContext
	}{
		{name: "nil context"},
		{name: "nil session info", data: &client.SessionContext{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := (&Session{SessionID: "legacy-session", Data: test.data}).ToProtobuf()
			if session.GetSessionId() != "legacy-session" {
				t.Fatalf("session_id = %q, want legacy-session", session.GetSessionId())
			}
			if session.GetTimer() == nil {
				t.Fatal("timer must be present for compatibility")
			}
		})
	}
}
