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
		logs.Log.Debugf("listener - session_upsert raw=%d keypair=%v",
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

// GetConnection 统一的连接获取/创建函数 (适用于 TCP pipeline)
// 每个 TCP 连接创建独立 Connection，避免一个 EOF 杀死共享 Connection。
func GetConnection(conn *cryptostream.Conn, pipelineID string, secureConfig *implanttypes.SecureConfig) (*Connection, error) {
	sid, err := cryptostream.PeekSid(conn)
	if err != nil {
		return nil, err
	}

	sessionID := hash.Md5Hash(encoders.Uint32ToBytes(sid))

	if existing := Connections.Get(sessionID); existing != nil {
		logs.Log.Debugf("connection - replace_existing session=%s raw=%d alive=%v pipeline=%s mode=tcp", sessionID, sid, existing.IsAlive(), pipelineID)
	}
	keyPair := GetKeyPairForSession(sid, secureConfig)
	newConn := NewConnection(conn.Parser, sid, pipelineID, keyPair)
	Connections.Add(newConn)
	return newConn, nil
}

// GetOrReuseConnection 复用已有连接 (适用于 HTTP simplex pipeline)
// HTTP 每个请求是短生命周期调用，不存在 EOF 互杀问题，
// 复用已有 Connection 让 cache 能跨请求积累命令。
func GetOrReuseConnection(conn *cryptostream.Conn, pipelineID string, secureConfig *implanttypes.SecureConfig) (*Connection, error) {
	sid, err := cryptostream.PeekSid(conn)
	if err != nil {
		return nil, err
	}

	sessionID := hash.Md5Hash(encoders.Uint32ToBytes(sid))

	if existing := Connections.Get(sessionID); existing != nil && existing.IsAlive() {
		existing.LastMessage = time.Now()
		return existing, nil
	}

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
		// secure is enabled but no keys established yet (cold start scenario).
		// Return an empty KeyPair so the parser enters "secure but plaintext" mode;
		// once key exchange completes and PushCtrl syncs the new keys, the parser
		// will pick them up on the next GetConnection / WithSecure call.
		return &clientpb.KeyPair{}
	}

	return &clientpb.KeyPair{
		PublicKey:  publicKey,
		PrivateKey: strings.Join(privateCandidates, "\n"),
	}
}

// Remove 移除 session
func (ls *listenerSessions) Remove(rawID uint32) {
	ls.sessions.Delete(rawID)
	logs.Log.Debugf("listener - session_remove raw=%d", rawID)
}

func NewConnection(p *parser.MessageParser, sid uint32, pipelineID string, keyPair *clientpb.KeyPair) *Connection {
	logs.Log.Debugf("connection - create raw=%d pipeline=%s keypair=%v", sid, pipelineID, keyPair != nil)

	// 如果有密钥对，创建安全的 parser
	if keyPair != nil {
		logs.Log.Debugf("connection - secure_enabled raw=%d pipeline=%s", sid, pipelineID)
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
			Connections.removeIfSame(c.SessionID, c)
		},
	)
}

func (c *Connection) closeWithError(err error) error {
	logs.Log.Debugf("connection - close session=%s raw=%d reason=%q", c.SessionID, c.RawID, err)
	c.fail(err)
	Connections.removeIfSame(c.SessionID, c)
	return err
}

func (c *Connection) runReceiveLoop() error {
	for c.IsAlive() {
		select {
		case req, ok := <-c.C:
			if !ok {
				return nil
			}
			logs.Log.Debugf("pipeline - receive_spite_request session=%s raw=%d name=%s", c.SessionID, c.RawID, req.Spite.Name)
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
		return c.closeWithError(fmt.Errorf("error reading header:%s %w", conn.RemoteAddr(), err))
	}
	GoGuarded("connection-send-call:"+c.SessionID, func() error {
		return c.Send(ctx, conn)
	}, c.runtimeErrorHandler("send call"))

	if err := c.buildResponse(conn, length); err != nil {
		return c.closeWithError(err)
	}
	return nil
}

func (c *Connection) HandlerSimplex(ctx context.Context, conn *cryptostream.Conn) error {
	var err error
	_, length, err := c.Parser.ReadHeader(conn)
	if err != nil {
		return c.closeWithError(fmt.Errorf("error reading header:%s %w", conn.RemoteAddr(), err))
	}
	if err := c.Send(ctx, conn); err != nil {
		return c.closeWithError(err)
	}
	if err := c.buildResponse(conn, length); err != nil {
		return c.closeWithError(err)
	}
	return nil
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
	select {
	case connect.C <- msg:
		return nil
	default:
		return fmt.Errorf("connection %s channel full", sid)
	}
}

//func (c *connections) GetFromRawID(rawID string) *Connection {
//	if val, ok := c.connections.Load(hash.Md5Hash([]byte(rawID))); ok {
//		return val.(*Connection)
//	}
//	return nil
//}

func (c *connections) Add(connect *Connection) *Connection {
	if connect == nil || connect.SessionID == "" {
		return connect
	}
	for {
		currentValue, loaded := c.connections.Load(connect.SessionID)
		if !loaded {
			actual, loaded := c.connections.LoadOrStore(connect.SessionID, connect)
			if !loaded {
				return connect
			}
			currentValue = actual
		}

		current, ok := currentValue.(*Connection)
		if !ok || current == connect {
			return connect
		}
		if current.IsAlive() && connect.LastMessage.Before(current.LastMessage) {
			logs.Log.Debugf("connection - stale_add_skip session=%s raw=%d current_raw=%d", connect.SessionID, connect.RawID, current.RawID)
			return current
		}
		if c.connections.CompareAndSwap(connect.SessionID, current, connect) {
			return connect
		}
	}
}

func (c *connections) Remove(sessionID string) {
	c.remove(sessionID, ErrConnectionRemoved)
}

func (c *connections) removeIfSame(sessionID string, conn *Connection) {
	if current := c.Get(sessionID); current == conn {
		c.connections.Delete(sessionID)
		logs.Log.Debugf("connection - map_remove session=%s", sessionID)
	} else {
		logs.Log.Debugf("connection - map_remove_skip session=%s reason=already_replaced", sessionID)
	}
}

func (c *connections) remove(sessionID string, err error) {
	conn := c.Get(sessionID)
	if conn != nil {
		conn.fail(err)
		c.connections.Delete(sessionID)
	}
}
