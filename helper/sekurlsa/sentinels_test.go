package sekurlsa

import (
	"errors"
	"fmt"
	"testing"
)

// TestSentinelErrors_AreDistinct guards the public error contract:
// each sentinel is its own value (errors.Is matches the named error
// but not its siblings). A regression where two sentinels alias the
// same error.New target would silently break callers that dispatch
// on error type.
//
// The 6 sentinels are part of the package's stable surface; this
// test fails-loud the moment one is renamed, deleted, or merged.
func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := map[string]error{
		"ErrNotMinidump":             ErrNotMinidump,
		"ErrUnsupportedBuild":        ErrUnsupportedBuild,
		"ErrLSASRVNotFound":          ErrLSASRVNotFound,
		"ErrMSVNotFound":             ErrMSVNotFound,
		"ErrKeyExtractFailed":        ErrKeyExtractFailed,
		"ErrUnsupportedArchitecture": ErrUnsupportedArchitecture,
	}
	for name, err := range sentinels {
		if err == nil {
			t.Errorf("%s is nil — exported sentinels must be non-nil", name)
			continue
		}
		if err.Error() == "" {
			t.Errorf("%s has empty message", name)
		}
	}
	// Pair-wise distinctness: errors.Is must distinguish each sentinel.
	for nameA, errA := range sentinels {
		for nameB, errB := range sentinels {
			if nameA == nameB {
				continue
			}
			if errors.Is(errA, errB) {
				t.Errorf("%s and %s alias the same error value", nameA, nameB)
			}
		}
	}
}

// TestSentinelErrors_WrapPropagation ensures errors.Is keeps working
// after a sentinel is wrapped with %w — the standard Parse() pattern
// is `fmt.Errorf("%w: build %d", ErrUnsupportedBuild, buildNum)`,
// callers expect errors.Is(err, ErrUnsupportedBuild) to still match.
func TestSentinelErrors_WrapPropagation(t *testing.T) {
	for _, sentinel := range []error{
		ErrNotMinidump,
		ErrUnsupportedBuild,
		ErrLSASRVNotFound,
		ErrMSVNotFound,
		ErrKeyExtractFailed,
		ErrUnsupportedArchitecture,
	} {
		wrapped := fmt.Errorf("context: %w", sentinel)
		if !errors.Is(wrapped, sentinel) {
			t.Errorf("errors.Is(wrapped, %v) = false, want true", sentinel)
		}
	}
}
