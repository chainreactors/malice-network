package core

import (
	"errors"
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
	inotify "github.com/chainreactors/malice-network/server/internal/notify"
)

func TestPrepareNotifyConfigTransaction(t *testing.T) {
	broker := &eventBroker{notifier: inotify.NewNotifier()}

	if _, err := broker.PrepareNotifyConfig(&configs.NotifyConfig{
		Enable:     true,
		ServerChan: &configs.ServerChanConfig{Enable: true, URL: "not-a-url"},
	}); err == nil {
		t.Fatal("expected a validation error")
	}

	commit, err := broker.PrepareNotifyConfig(&configs.NotifyConfig{Enable: false})
	if err != nil {
		t.Fatalf("PrepareNotifyConfig failed: %v", err)
	}
	commit()

	for _, unavailable := range []*eventBroker{nil, {}} {
		if _, err := unavailable.PrepareNotifyConfig(&configs.NotifyConfig{}); !errors.Is(err, ErrEventBrokerUnavailable) {
			t.Fatalf("unavailable broker prepare: %v", err)
		}
		if err := unavailable.InitService(&configs.NotifyConfig{}); !errors.Is(err, ErrEventBrokerUnavailable) {
			t.Fatalf("unavailable broker init: %v", err)
		}
	}
}
