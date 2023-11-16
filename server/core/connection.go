package core

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"google.golang.org/protobuf/proto"
	"net"
	"sync"
	"time"
)

var (
	Connections = &connections{
		connections: &sync.Map{},
	}
)

type spitesCache struct {
	cache []*commonpb.Spite
	max   int
}

func (sc spitesCache) Len() int {
	return len(sc.cache)
}

func (sc spitesCache) Build() *commonpb.Spites {
	spites := &commonpb.Spites{Spites: []*commonpb.Spite{}}
	for _, s := range sc.cache {
		spites.Spites = append(spites.Spites, s)
	}
	spites.Reset()
	return spites
}

func (sc spitesCache) Reset() {
	sc.cache = []*commonpb.Spite{}
}

func (sc spitesCache) Append(spite *commonpb.Spite) {
	sc.cache = append(sc.cache, spite)
}

func NewConnection(rawid []byte) *Connection {
	conn := &Connection{
		RawID:       rawid,
		SessionID:   hash.Md5Hash(rawid),
		LastMessage: time.Now(),
		C:           make(chan proto.Message, 255),
		Sender:      make(chan *commonpb.Spites, 1),
		Alive:       true,
	}
	Connections.Add(conn)
	var spites spitesCache
	go func() {
		for {
			select {
			case spite := <-conn.C:
				spites.Append(spite.(*commonpb.Spite))
			}
		}
	}()

	go func() {
		for {
			if spites.Len() == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			select {
			case conn.Sender <- spites.Build():
			}
		}
	}()
	return conn
}

type Connection struct {
	RawID       []byte
	SessionID   string
	LastMessage time.Time
	C           chan proto.Message // spite
	Sender      chan *commonpb.Spites
	Alive       bool
	lock        sync.RWMutex
}

func (c *Connection) Send(conn net.Conn) {
	msg := <-c.Sender
	err := packet.WritePacket(conn, msg, c.RawID)
	if err != nil {
		// retry
		logs.Log.Debugf("Error write packet, %s", err.Error())
		c.Sender <- msg
		return
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
