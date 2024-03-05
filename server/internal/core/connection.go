package core

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"net"
	"sync"
	"time"
)

var (
	Connections = &connections{
		connections: &sync.Map{},
	}
)

func NewConnection(rawid []byte) *Connection {
	conn := &Connection{
		RawID:       rawid,
		SessionID:   hash.Md5Hash(rawid),
		LastMessage: time.Now(),
		C:           make(chan *implantpb.Spite, 255),
		Sender:      make(chan *implantpb.Spites, 1),
		Alive:       true,
		cache:       types.NewSpitesCache(),
	}
	Connections.Add(conn)
	go func() {
		for {
			select {
			case spite := <-conn.C:
				conn.cache.Append(spite)
			}
		}
	}()

	go func() {
		for {
			if conn.cache.Len() == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			select {
			case conn.Sender <- conn.cache.Build():
			}
		}
	}()
	return conn
}

type Connection struct {
	RawID       []byte
	SessionID   string
	LastMessage time.Time
	C           chan *implantpb.Spite // spite
	Sender      chan *implantpb.Spites
	Alive       bool
	cache       *types.SpitesCache
}

func (c *Connection) Send(ctx context.Context, conn net.Conn) {
	select {
	case <-time.After(100 * time.Millisecond):
		return
	case <-ctx.Done():
		return
	case msg := <-c.Sender:
		err := packet.WritePacket(conn, msg, c.RawID)
		if err != nil {
			// retry
			logs.Log.Debugf("Error write packet, %s", err.Error())
			c.Sender <- msg
			return
		}
	}
}

type connections struct {
	connections *sync.Map // map[session_id]*Session
}

func (c *connections) All() []*Connection {
	all := []*Connection{}
	c.connections.Range(func(key, value interface{}) bool {
		all = append(all, value.(*Connection))
		return true
	})
	return all
}

func (c *connections) Get(sessionID string) *Connection {
	if val, ok := c.connections.Load(sessionID); ok {
		return val.(*Connection)
	}
	return nil
}

//func (c *connections) GetFromRawID(rawID string) *Connection {
//	if val, ok := c.connections.Load(hash.Md5Hash([]byte(rawID))); ok {
//		return val.(*Connection)
//	}
//	return nil
//}

// Add - Add a sliver to the hive (atomically)
func (c *connections) Add(connect *Connection) *Connection {
	c.connections.Store(connect.SessionID, connect)
	//EventBroker.Publish(Event{
	//	EventType: consts.SessionOpenedEvent,
	//	Session:   session,
	//})
	return connect
}

func (c *connections) Remove(sessionID string) {
	conn := c.Get(sessionID)
	conn.Alive = false
}
