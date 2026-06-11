package build

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

type errReader struct {
	data []byte
	err  error
	read bool
}

func (r *errReader) Read(p []byte) (int, error) {
	if !r.read {
		r.read = true
		n := copy(p, r.data)
		if n > 0 {
			return n, nil
		}
	}
	return 0, r.err
}

func TestConsumeLogPipeSanitizesControlBytes(t *testing.T) {
	oldUpdate := updateBuilderLog
	defer func() { updateBuilderLog = oldUpdate }()

	var lines []string
	updateBuilderLog = func(name string, line string) {
		lines = append(lines, line)
	}

	err := consumeLogPipe(strings.NewReader("ok\x01line\nnext\n"), "artifact-a")
	if err != nil {
		t.Fatalf("consumeLogPipe failed: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2", len(lines))
	}
	if lines[0] != "okline\n" {
		t.Fatalf("sanitized line = %q, want %q", lines[0], "okline\n")
	}
}

func TestConsumeLogPipeReturnsScannerError(t *testing.T) {
	want := errors.New("scan failed")
	err := consumeLogPipe(&errReader{err: want}, "artifact-b")
	if !errors.Is(err, want) {
		t.Fatalf("consumeLogPipe error = %v, want %v", err, want)
	}
}

func TestRunLogWorkerReturnsErrorAndRunsCleanup(t *testing.T) {
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	cleaned := false
	want := errors.New("worker failed")

	runLogWorker(&wg, errCh, "worker-test", func() error {
		return want
	}, func() {
		cleaned = true
	})

	wg.Wait()
	close(errCh)

	if !cleaned {
		t.Fatal("cleanup did not run")
	}

	err := <-errCh
	if !errors.Is(err, want) {
		t.Fatalf("worker error = %v, want %v", err, want)
	}
}

func TestRunLogWorkerIgnoresEOFScannerEquivalent(t *testing.T) {
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	runLogWorker(&wg, errCh, "worker-eof", func() error {
		return io.EOF
	})

	wg.Wait()
	close(errCh)

	err := <-errCh
	if !errors.Is(err, io.EOF) {
		t.Fatalf("worker error = %v, want %v", err, io.EOF)
	}
}
