package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	types "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/implanttypes"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/server/internal/parser"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
)

var (
	Connections = &connections{
		connections: &sync.Map{},
	}
	ListenerSessions = &listenerSessions{
		sessions: &sync.Map{},
	}
	ErrConnectionRemoved = fmt.Errorf("connection removed")
)

// listenerSessions 管理 listener 端的 session 信息
type listenerSessions struct {
	sessions *sync.Map // map[uint32]*clientpb.Session
}

// Add 添加或更新 session
func (ls *listenerSessions) Add(session *clientpb.Session) {
	if session != nil {
		ls.sessions.Store(session.RawId, session)
		logs.Log.Debugf("[listener] added/updated session %d with KeyPair: %v",
			session.RawId, session.KeyPair != nil)
	}
}

// Get 获取 session
func (ls *listenerSessions) Get(rawID uint32) *clientpb.Session {
	if val, ok := ls.sessions.Load(rawID); ok {
		return val.(*clientpb.Session)
	}
	return nil
}

// GetConnection 统一的连接获取/创建函数 (适用于 TCP 和 HTTP pipeline)
// 从 cryptostream.Conn 中提取 SID 并获取/创建连接
func GetConnection(conn *cryptostream.Conn, pipelineID string, secureConfig *implanttypes.SecureConfig) (*Connection, error) {
	sid, err := cryptostream.PeekSid(conn)
	if err != nil {
		return nil, err
	}

	sessionID := hash.Md5Hash(encoders.Uint32ToBytes(sid))

	// 尝试从现有连接池获取连接
	if existingConn := Connections.Get(sessionID); existingConn != nil {
		// 获取 KeyPair 并更新现有连接的安全配置
		keyPair := GetKeyPairForSession(sid, secureConfig)
		if keyPair != nil {
			existingConn.Parser.WithSecure(keyPair)
		}
		return existingConn, nil
	}

	// 创建新连接
	keyPair := GetKeyPairForSession(sid, secureConfig)
	newConn := NewConnection(conn.Parser, sid, pipelineID, keyPair)
	Connections.Add(newConn)
	return newConn, nil
}

// GetKeyPairForSession 获取会话的密钥对
// 优先从 ListenerSessions 获取，如果没有则从 secureConfig 获取交换密钥对
func GetKeyPairForSession(sid uint32, secureConfig *implanttypes.SecureConfig) *clientpb.KeyPair {
	// 优先从 session 中获取 KeyPair
	if secureConfig == nil || !secureConfig.Enable {
		return nil
	}

	var sessionKeyPair *clientpb.KeyPair
	if session := ListenerSessions.Get(sid); session != nil {
		sessionKeyPair = session.KeyPair
	}

	// 组装解密私钥候选：优先当前会话私钥，回退 pipeline server 私钥。
	privateCandidates := make([]string, 0, 2)
	appendPrivate := func(key string) {
		key = strings.TrimSpace(key)
		if key == "" {
			return
		}
		for _, existing := range privateCandidates {
			if existing == key {
				return
			}
		}
		privateCandidates = append(privateCandidates, key)
	}

	if sessionKeyPair != nil {
		appendPrivate(sessionKeyPair.PrivateKey)
	}
	appendPrivate(secureConfig.ServerPrivateKey)

	publicKey := secureConfig.ImplantPublicKey
	if sessionKeyPair != nil && strings.TrimSpace(sessionKeyPair.PublicKey) != "" {
		publicKey = strings.TrimSpace(sessionKeyPair.PublicKey)
	}

	if publicKey == "" && len(privateCandidates) == 0 {
		return nil
	}

	return &clientpb.KeyPair{
		PublicKey:  publicKey,
		PrivateKey: strings.Join(privateCandidates, "\n"),
	}
}

// Remove 移除 session
func (ls *listenerSessions) Remove(rawID uint32) {
	ls.sessions.Delete(rawID)
	logs.Log.Debugf("[listener] removed session %d", rawID)
}

func NewConnection(p *parser.MessageParser, sid uint32, pipelineID string, keyPair *clientpb.KeyPair) *Connection {
	logs.Log.Debugf("[connection] creating connection %d with KeyPair: %v", sid, keyPair != nil)

	// 如果有密钥对，创建安全的 parser
	if keyPair != nil {
		logs.Log.Debugf("[connection] enabled secure mode for connection %d", sid)
		p.WithSecure(keyPair)
	}

	conn := &Connection{
		PipelineID:  pipelineID,
		RawID:       sid,
		SessionID:   hash.Md5Hash(encoders.Uint32ToBytes(sid)),
		LastMessage: time.Now(),
		C:           make(chan *clientpb.SpiteRequest, 255),
		Sender:      make(chan *implantpb.Spites, 1),
		cache:       parser.NewSpitesBuf(),
		Parser:      p,
	}
	conn.alive.Store(true)

	GoGuarded("connection-recv:"+conn.SessionID, conn.runReceiveLoop, conn.runtimeErrorHandler("receive loop"))
	GoGuarded("connection-send:"+conn.SessionID, conn.runSenderLoop, conn.runtimeErrorHandler("sender loop"))
	return conn
}

type Connection struct {
	RawID       uint32
	SessionID   string
	LastMessage time.Time
	PipelineID  string
	C           chan *clientpb.SpiteRequest // spite
	Sender      chan *implantpb.Spites
	Parser      *parser.MessageParser
	cache       *parser.SpitesCache
	alive       atomic.Bool
	errMu       sync.Mutex
	lastErr     error
}

func (c *Connection) IsAlive() bool {
	return c.alive.Load()
}

func (c *Connection) LastError() error {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	return c.lastErr
}

func (c *Connection) fail(err error) {
	if err != nil {
		c.errMu.Lock()
		if c.lastErr == nil {
			c.lastErr = err
		}
		c.errMu.Unlock()
	}
	c.alive.Store(false)
}

func (c *Connection) runtimeErrorHandler(scope string) GoErrorHandler {
	label := fmt.Sprintf("connection %s %s", c.SessionID, scope)
	return CombineErrorHandlers(
		LogGuardedError(label),
		func(err error) {
			c.fail(err)
			Connections.remove(c.SessionID, err)
		},
	)
}

func (c *Connection) runReceiveLoop() error {
	for c.IsAlive() {
		select {
		case req, ok := <-c.C:
			if !ok {
				return nil
			}
			logs.Log.Debugf("[pipeline] received spite_request %s", req.Spite.Name)
			c.cache.Append(req.Spite)
		case <-time.After(100 * time.Millisecond):
		}
	}
	return nil
}

func (c *Connection) runSenderLoop() error {
	for c.IsAlive() {
		if c.cache.Len() == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		select {
		case c.Sender <- c.cache.Build():
		case <-time.After(100 * time.Millisecond):
		}
	}
	return nil
}

func (c *Connection) Send(ctx context.Context, conn *cryptostream.Conn) error {
	select {
	case <-time.After(1000 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return nil
	case msg, ok := <-c.Sender:
		if !ok || msg == nil {
			return nil
		}
		// Parser 内部会自动处理加解密逻辑
		err := c.Parser.WritePacket(conn, msg, c.RawID)
		if err != nil {
			return fmt.Errorf("write packet for connection %s: %w", c.SessionID, err)
		}
	}
	return nil
}

func (c *Connection) buildResponse(conn *cryptostream.Conn, length uint32) error {
	var msg *implantpb.Spites
	if length >= 2 {
		var err error
		msg, err = c.Parser.ReadMessage(conn, length)
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
	GoGuarded("connection-send-call:"+c.SessionID, func() error {
		return c.Send(ctx, conn)
	}, c.runtimeErrorHandler("send call"))

	return c.buildResponse(conn, length)
}

func (c *Connection) HandlerSimplex(ctx context.Context, conn *cryptostream.Conn) error {
	var err error
	_, length, err := c.Parser.ReadHeader(conn)
	if err != nil {
		return fmt.Errorf("error reading header:%s %w", conn.RemoteAddr(), err)
	}
	if err := c.Send(ctx, conn); err != nil {
		return err
	}
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
	if !connect.IsAlive() {
		return fmt.Errorf("connection %s is not alive", sid)
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
	c.remove(sessionID, ErrConnectionRemoved)
}

func (c *connections) remove(sessionID string, err error) {
	conn := c.Get(sessionID)
	if conn != nil {
		conn.fail(err)
		c.connections.Delete(sessionID)
	}
}
