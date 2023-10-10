package core

import (
	"github.com/chainreactors/malice-network/utils/encoders"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

func NewConnection(listenerID, remoteAddr string) *Connection {
	return &Connection{
		ID:          encoders.UUID(),
		Forwarder:   Forwarders.Get(listenerID),
		RemoteAddr:  remoteAddr,
		LastMessage: time.Now(),
		Sender:      make(chan proto.Message, 255),
		RespMap:     new(sync.Map),
	}
}

type Connection struct {
	ID          string
	RemoteAddr  string
	LastMessage time.Time
	Sender      chan proto.Message
	RespMap     *sync.Map
	Forwarder   *Forward
}
