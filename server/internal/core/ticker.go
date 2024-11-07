package core

import (
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
)

var (
	GlobalTicker = NewTicker()
	mutex        sync.Mutex
)

type Ticker struct {
	cron *cron.Cron
}

func NewTicker() *Ticker {
	mutex.Lock()
	defer mutex.Unlock()

	ticker := &Ticker{
		cron: cron.New(),
	}
	ticker.cron.Start()
	return ticker
}

func (t *Ticker) Start(interval int, cmd func()) (cron.EntryID, error) {
	return t.cron.AddFunc(fmt.Sprintf("@every %ds", interval), cmd)
}

func (t *Ticker) Remove(id cron.EntryID) {
	t.cron.Remove(id)
}

func (t *Ticker) RemoveAll() {
	t.cron.Stop()
}
