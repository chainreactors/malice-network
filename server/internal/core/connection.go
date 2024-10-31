package core

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/parser"
	"sync"
	"time"
)

var (
	Connections = &connections{
		connections: &sync.Map{},
	}
)

func NewConnection(p *parser.MessageParser, sid []byte, pipelineID string) *Connection {
	conn := &Connection{
		PipelineID:  pipelineID,
		RawID:       sid,
		SessionID:   hash.Md5Hash(sid),
		LastMessage: time.Now(),
		C:           make(chan *clientpb.SpiteRequest, 255),
		Sender:      make(chan *implantpb.Spites, 1),
		Alive:       true,
		cache:       parser.NewSpitesBuf(),
		Parser:      p,
	}

	go func() {
		for {
			select {
			case req := <-conn.C:
				logs.Log.Debugf("Received spite %s", req.Spite.Name)
				conn.cache.Append(req.Spite)
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
	PipelineID  string
	C           chan *clientpb.SpiteRequest // spite
	Sender      chan *implantpb.Spites
	Alive       bool
	Parser      *parser.MessageParser
	cache       *parser.SpitesCache
}

func (c *Connection) Send(ctx context.Context, conn *peek.Conn) {
	select {
	case <-time.After(100 * time.Millisecond):
		return
	case <-ctx.Done():
		return
	case msg := <-c.Sender:
		err := c.Parser.WritePacket(conn, msg, c.RawID)
		if err != nil {
			// retry
			logs.Log.Debugf("Error write packet, %s", err.Error())
			c.Sender <- msg
			return
		}
	}
}

func (c *Connection) Handler(ctx context.Context, conn *peek.Conn) error {
	var err error
	_, length, err := c.Parser.ReadHeader(conn)
	if err != nil {
		//logs.Log.Debugf("Error reading header: %s %v", conn.RemoteAddr(), err)
		return fmt.Errorf("error reading header:%s %w", conn.RemoteAddr(), err)
	}
	var msg *implantpb.Spites
	go c.Send(ctx, conn)
	if length != 1 {
		msg, err = c.Parser.ReadMessage(conn, length)
		if err != nil {
			return fmt.Errorf("error reading message:%s %w", conn.RemoteAddr(), err)
		}
		if msg.Spites == nil {
			msg = types.BuildPingSpite()
		}
	} else {
		msg = types.BuildPingSpite()
	}

	Forwarders.Send(c.PipelineID, &Message{
		Spites:     msg,
		SessionID:  c.SessionID,
		RemoteAddr: conn.RemoteAddr().String(),
	})
	return nil
}

type connections struct {
	connections *sync.Map // map[session_id]*Session
}

func (c *connections) NeedConnection(conn *peek.Conn, pipelineID string) (*Connection, error) {
	p, err := parser.NewParser(conn)
	if err != nil {
		return nil, err
	}
	sid, _, err := p.PeekHeader(conn)
	if err != nil {
		return nil, err
	}

	if newC := c.Get(hash.Md5Hash(sid)); newC != nil {
		return newC, nil
	} else {
		newC := NewConnection(p, sid, pipelineID)
		c.Add(newC)
		return newC, nil
	}
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

func (c *connections) Add(connect *Connection) *Connection {
	c.connections.Store(connect.SessionID, connect)
	return connect
}

func (c *connections) Remove(sessionID string) {
	conn := c.Get(sessionID)
	conn.Alive = false
}
