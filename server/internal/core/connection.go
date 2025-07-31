package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/parser"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
)

var (
	Connections = &connections{
		connections: &sync.Map{},
	}
)

func NewConnection(p *parser.MessageParser, sid uint32, pipelineID string, keyPair *clientpb.KeyPair) *Connection {
	logs.Log.Debugf("[connection] creating connection %d with KeyPair: %v", sid, keyPair != nil)

	conn := &Connection{
		PipelineID:  pipelineID,
		RawID:       sid,
		SessionID:   hash.Md5Hash(encoders.Uint32ToBytes(sid)),
		LastMessage: time.Now(),
		C:           make(chan *clientpb.SpiteRequest, 255),
		Sender:      make(chan *implantpb.Spites, 1),
		Alive:       true,
		cache:       parser.NewSpitesBuf(),
		Parser:      p,
		keyPair:     keyPair,
	}

	go func() {
		for {
			select {
			case req := <-conn.C:
				logs.Log.Debugf("[pipeline] received spite_request %s", req.Spite.Name)
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
	RawID       uint32
	SessionID   string
	LastMessage time.Time
	PipelineID  string
	C           chan *clientpb.SpiteRequest // spite
	Sender      chan *implantpb.Spites
	Alive       bool
	Parser      *parser.MessageParser
	cache       *parser.SpitesCache

	// Age 密钥对（来自 ListenerSessions）
	keyPair *clientpb.KeyPair
}

func (c *Connection) Send(ctx context.Context, conn *cryptostream.Conn) {
	select {
	case <-time.After(1000 * time.Millisecond):
		return
	case <-ctx.Done():
		return
	case msg := <-c.Sender:
		var err error

		// 检查是否启用安全模式（有密钥对）
		if c.keyPair != nil {
			// 使用安全写入（基于 Age 密钥对）
			err = c.Parser.SecureWritePacket(conn, msg, c.RawID, c.keyPair)
		} else {
			// 使用普通写入
			err = c.Parser.WritePacket(conn, msg, c.RawID)
		}

		if err != nil {
			// retry
			logs.Log.Debugf("Error write packet, %s", err.Error())
			c.Sender <- msg
			return
		}
	}
}

func (c *Connection) buildResponse(conn *cryptostream.Conn, length uint32) error {
	var msg *implantpb.Spites
	if length >= 2 {
		var err error

		// 检查是否启用安全模式（有密钥对）
		if c.keyPair != nil {
			// 使用安全读取（基于 Age 密钥对）
			_, msg, err = c.Parser.SecureReadPacket(conn, c.keyPair)
		} else {
			// 使用普通读取
			msg, err = c.Parser.ReadMessage(conn, length)
		}

		if err != nil {
			return fmt.Errorf("error reading message:%s %w", conn.RemoteAddr(), err)
		}
		if msg.Spites == nil {
			msg = types.BuildPingSpites()
		}
	} else {
		msg = types.BuildPingSpites()
	}

	Forwarders.Send(c.PipelineID, &Message{
		Spites:     msg,
		SessionID:  c.SessionID,
		RawID:      c.RawID,
		RemoteAddr: conn.RemoteAddr().String(),
	})
	return nil
}

func (c *Connection) Handler(ctx context.Context, conn *cryptostream.Conn) error {
	var err error
	_, length, err := c.Parser.ReadHeader(conn)
	if err != nil {
		return fmt.Errorf("error reading header:%s %w", conn.RemoteAddr(), err)
	}
	go c.Send(ctx, conn)

	return c.buildResponse(conn, length)
}

func (c *Connection) HandlerSimplex(ctx context.Context, conn *cryptostream.Conn) error {
	var err error
	_, length, err := c.Parser.ReadHeader(conn)
	if err != nil {
		return fmt.Errorf("error reading header:%s %w", conn.RemoteAddr(), err)
	}
	c.Send(ctx, conn)
	return c.buildResponse(conn, length)
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

func (c *connections) Push(sid string, msg *clientpb.SpiteRequest) error {
	connect := Connections.Get(sid)
	if connect == nil {
		return fmt.Errorf("connection %s not found", sid)
	}
	connect.C <- msg
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
