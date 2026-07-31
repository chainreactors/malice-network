package server

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRegisterSaasLicenseInBackgroundReturnsBeforeRegistrationCompletes(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	errCh := registerSaasLicenseInBackground(time.Second, func(context.Context) error {
		close(started)
		<-release
		return nil
	})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("background registration did not start")
	}

	select {
	case <-errCh:
		t.Fatal("background registration finished before it was released")
	default:
	}

	close(release)
	select {
	case err, ok := <-errCh:
		if ok || err != nil {
			t.Fatalf("background registration result = %v, open = %v; want closed channel", err, ok)
		}
	case <-time.After(time.Second):
		t.Fatal("background registration did not finish")
	}
}

func TestRegisterSaasLicenseInBackgroundAppliesTimeout(t *testing.T) {
	errCh := registerSaasLicenseInBackground(20*time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	select {
	case err := <-errCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("background registration error = %v, want context deadline exceeded", err)
		}
	case <-time.After(time.Second):
		t.Fatal("background registration did not time out")
	}
}
