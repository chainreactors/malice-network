package server

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/logs"
)

func TestConfigureDebugLoggerFormat(t *testing.T) {
	logger := logs.NewLogger(logs.DebugLevel)
	var out bytes.Buffer
	logger.SetOutput(&out)

	now := func() time.Time {
		return time.Date(2026, 5, 13, 21, 36, 28, 987000000, time.Local)
	}
	configureDebugLogger(logger, now)

	logger.Debugf("crypto.wrap - encryption_configs_count=%d", 2)
	logger.Infof("event.job - Job %d %d website_start:", 7, 1)
	logger.Errorf("connection - close session=%s raw=%d reason=%q", "0123456789abcdef0123456789abcdef", 801492628, "EOF")

	got := out.String()
	wantLines := []string{
		"[05.13 21:36:28] DBG crypto.wrap - encryption_configs_count=2",
		"[05.13 21:36:28] INF event.job - Job 7 1 website_start:",
		"[05.13 21:36:28] ERR connection - close session=0123456789abcdef0123456789abcdef raw=801492628 reason=\"EOF\"",
	}
	for _, want := range wantLines {
		if !strings.Contains(got, want) {
			t.Fatalf("formatted log missing %q in:\n%s", want, got)
		}
	}

	if strings.Contains(got, "[debug]") {
		t.Fatalf("formatted log should not contain old debug label:\n%s", got)
	}
	if strings.Contains(got, ".987") {
		t.Fatalf("formatted log should be second precision:\n%s", got)
	}
}
