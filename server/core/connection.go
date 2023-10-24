package core

import (
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

var (
	Connections = &connections{
		connections: &sync.Map{},
	}
)

func NewConnection(rawid string) *Connection {
	conn := &Connection{
		RawID:       rawid,
		SessionID:   hash.Md5Hash([]byte(rawid)),
		LastMessage: time.Now(),
		Sender:      make(chan proto.Message, 255),
		Alive:       true,
	}
	Connections.Add(conn)
	return conn
}

type Connection struct {
	RawID       string
	SessionID   string
	LastMessage time.Time
	Sender      chan proto.Message // spite/promise
	Alive       bool
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
