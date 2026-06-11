package rpc

import (
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/types"
)

func TestMissingRequestErrorsUseDistinctMessages(t *testing.T) {
	if got := types.ErrMissingRequestField.Error(); !strings.Contains(strings.ToLower(got), "missing required request field") {
		t.Fatalf("ErrMissingRequestField = %q, want generic missing required request field", got)
	}
	if got := types.ErrMissingSessionRequestField.Error(); !strings.Contains(strings.ToLower(got), "missing session request field") {
		t.Fatalf("ErrMissingSessionRequestField = %q, want session-specific message", got)
	}
}
