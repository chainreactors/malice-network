package socks5

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
)

// Status values for a local SOCKS listener record.
const (
	StatusStarting  = "starting"
	StatusListening = "listening"
	StatusDegraded  = "degraded"
	StatusStopping  = "stopping"

	// openFailRebuildThreshold clears the cached relay task so the next
	// ensureSessionRelay issues a fresh TcpRelay START after repeated failures.
	openFailRebuildThreshold = 5
	// maxDataQChunks caps ordered downlink chunks per connection (backpressure).
	maxDataQChunks = 256
	// closeWaitTimeout bounds SocksService.Close waiting on in-flight handlers.
	closeWaitTimeout = 300 * time.Millisecond
)

// registry holds all SOCKS listeners in this client process (cross-session).
var registry = newSocksRegistry()

type socksRegistry struct {
	mu       sync.Mutex
	byKey    map[string]*SocksService            // sessionID|bind|port
	byID     map[string]*SocksService            // id
	bySess   map[string]map[string]*SocksService // sessionID -> key -> svc
	relays   map[string]*sessionRelay            // sessionID -> shared tcp_relay
}

func newSocksRegistry() *socksRegistry {
	return &socksRegistry{
		byKey:  make(map[string]*SocksService),
		byID:   make(map[string]*SocksService),
		bySess: make(map[string]map[string]*SocksService),
		relays: make(map[string]*sessionRelay),
	}
}

func serviceKey(sessionID, bind string, port int) string {
	return fmt.Sprintf("%s|%s|%d", sessionID, bind, port)
}

func shortID(sessionID, bind string, port int) string {
	sid := sessionID
	if len(sid) > 8 {
		sid = sid[:8]
	}
	return fmt.Sprintf("%s-%s-%d", sid, bind, port)
}

// sessionRelay is one shared tcp_relay long-task per implant session.
type sessionRelay struct {
	mu                sync.Mutex
	sessionID         string
	task              *clientpb.Task
	nextID            atomic.Uint32
	callbackInstalled bool
	openFailStreak    int
	lastError         string
	// owners routes tunnel events by conn_id to the listener that opened it.
	owners map[uint32]*SocksService
	// pendingOpen buffers OpenResult that races ahead of registerConn.
	pendingOpen map[uint32]*implantpb.TunnelOpenResult
}

func (r *socksRegistry) getRelay(sessionID string) *sessionRelay {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rel, ok := r.relays[sessionID]; ok {
		return rel
	}
	rel := &sessionRelay{
		sessionID:   sessionID,
		owners:      make(map[uint32]*SocksService),
		pendingOpen: make(map[uint32]*implantpb.TunnelOpenResult),
	}
	rel.nextID.Store(1)
	r.relays[sessionID] = rel
	return rel
}

// registerConn binds connID to svc. Returns a buffered OpenResult if one arrived early.
func (rel *sessionRelay) registerConn(connID uint32, svc *SocksService) *implantpb.TunnelOpenResult {
	rel.mu.Lock()
	defer rel.mu.Unlock()
	if rel.owners == nil {
		rel.owners = make(map[uint32]*SocksService)
	}
	rel.owners[connID] = svc
	if rel.pendingOpen != nil {
		if p := rel.pendingOpen[connID]; p != nil {
			delete(rel.pendingOpen, connID)
			return p
		}
	}
	return nil
}

func (rel *sessionRelay) unregisterConn(connID uint32) {
	rel.mu.Lock()
	defer rel.mu.Unlock()
	if rel.owners != nil {
		delete(rel.owners, connID)
	}
	if rel.pendingOpen != nil {
		delete(rel.pendingOpen, connID)
	}
}

func (rel *sessionRelay) ownerOf(connID uint32) *SocksService {
	rel.mu.Lock()
	defer rel.mu.Unlock()
	if rel.owners == nil {
		return nil
	}
	return rel.owners[connID]
}

func (r *socksRegistry) addService(svc *SocksService) error {
	key := serviceKey(svc.sessionID, svc.bind, svc.port)
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byKey[key]; ok {
		return fmt.Errorf("socks5 already listening on %s:%d for session %s", svc.bind, svc.port, shortSession(svc.sessionID))
	}
	// Port uniqueness in this process (any session): avoid bind clash confusion.
	for _, other := range r.byKey {
		if other.port == svc.port && (other.bind == svc.bind || other.bind == "0.0.0.0" || svc.bind == "0.0.0.0") {
			return fmt.Errorf("port %d already in use by session %s (%s:%d)", svc.port, shortSession(other.sessionID), other.bind, other.port)
		}
	}
	r.byKey[key] = svc
	r.byID[svc.id] = svc
	if r.bySess[svc.sessionID] == nil {
		r.bySess[svc.sessionID] = make(map[string]*SocksService)
	}
	r.bySess[svc.sessionID][key] = svc
	return nil
}

func (r *socksRegistry) removeService(svc *SocksService) (sessionEmpty bool) {
	key := serviceKey(svc.sessionID, svc.bind, svc.port)
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byKey, key)
	delete(r.byID, svc.id)
	if m := r.bySess[svc.sessionID]; m != nil {
		delete(m, key)
		if len(m) == 0 {
			delete(r.bySess, svc.sessionID)
			return true
		}
	}
	return false
}

func (r *socksRegistry) list(filterSession string) []*SocksService {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*SocksService, 0, len(r.byKey))
	for _, svc := range r.byKey {
		if filterSession != "" && svc.sessionID != filterSession {
			continue
		}
		out = append(out, svc)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].sessionID != out[j].sessionID {
			return out[i].sessionID < out[j].sessionID
		}
		if out[i].port != out[j].port {
			return out[i].port < out[j].port
		}
		return out[i].bind < out[j].bind
	})
	return out
}

func (r *socksRegistry) getByPort(sessionID string, port int) *SocksService {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, svc := range r.byKey {
		if svc.sessionID == sessionID && svc.port == port {
			return svc
		}
	}
	return nil
}

func (r *socksRegistry) getByID(id string) *SocksService {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.byID[id]
}

func (r *socksRegistry) sessionServices(sessionID string) []*SocksService {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := r.bySess[sessionID]
	out := make([]*SocksService, 0, len(m))
	for _, svc := range m {
		out = append(out, svc)
	}
	return out
}

// routeSpite delivers tunnel events to the owning listener only (by conn_id).
// OpenResult without an owner is buffered on the session relay until registerConn.
func (r *socksRegistry) routeSpite(sessionID string, spite *implantpb.Spite) {
	if spite == nil {
		return
	}
	var connID uint32
	var hasID bool
	if x := spite.GetTunnelOpenResult(); x != nil {
		connID, hasID = x.ConnId, true
	} else if x := spite.GetTunnelData(); x != nil {
		connID, hasID = x.ConnId, true
	} else if x := spite.GetTunnelClose(); x != nil {
		connID, hasID = x.ConnId, true
	}
	if !hasID {
		return
	}

	rel := r.getRelay(sessionID)
	if svc := rel.ownerOf(connID); svc != nil {
		svc.handleSpite(spite)
		return
	}

	// No owner yet: only OpenResult is worth buffering (Data/Close are dropped).
	if open := spite.GetTunnelOpenResult(); open != nil {
		rel.mu.Lock()
		if rel.pendingOpen == nil {
			rel.pendingOpen = make(map[uint32]*implantpb.TunnelOpenResult)
		}
		// Hard cap to avoid unbounded growth if clients disappear.
		if len(rel.pendingOpen) < 1024 {
			rel.pendingOpen[connID] = open
		}
		rel.mu.Unlock()
	}
}

// fanoutSpite is kept as a thin alias for older call sites/tests.
func (r *socksRegistry) fanoutSpite(sessionID string, spite *implantpb.Spite) {
	r.routeSpite(sessionID, spite)
}

func (r *socksRegistry) clearRelay(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.relays, sessionID)
}

func shortSession(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// SocksService is one local SOCKS5 listener bound to a session.
type SocksService struct {
	id        string
	sessionID string
	rpc       clientrpc.MaliceRPCClient
	sess      *client.Session
	con       *core.Console
	relay     *sessionRelay

	listener net.Listener
	user     string
	pass     string
	bind     string
	port     int

	status    string
	lastError string
	createdAt time.Time

	mu    sync.Mutex
	conns map[uint32]*localConn

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type localConn struct {
	id     uint32
	conn   net.Conn
	openCh chan *implantpb.TunnelOpenResult
	// Ordered byte stream from implant. Never drop (TLS requires integrity).
	dataMu   sync.Mutex
	dataQ    [][]byte
	dataWait chan struct{}
	closed   atomic.Bool
}

func Commands(con *core.Console) []*cobra.Command {
	cmd := &cobra.Command{
		Use:   consts.CommandSocks5,
		Short: "Native SOCKS5 proxy through implant (no REM)",
		Long:  "Start local SOCKS5 listener(s) that tunnel TCP CONNECT via the implant tcp_relay module. Multiple listeners per session are allowed and share one tcp_relay task. Requires username/password. Domain resolution is performed on the implant.",
		Annotations: map[string]string{
			"depend": consts.ModuleTcpRelay,
		},
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a local SOCKS5 listener (multiple per session OK)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StartCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": consts.ModuleTcpRelay,
		},
		Example: `~~~
socks5 start --port 1080 --user admin --pass secret
socks5 start --port 1081 --user bob --pass secret --bind 127.0.0.1
~~~`,
	}
	startCmd.Flags().IntP("port", "p", 1080, "local listen port")
	startCmd.Flags().String("bind", "127.0.0.1", "local bind address")
	startCmd.Flags().String("user", "", "SOCKS5 username (required)")
	startCmd.Flags().String("pass", "", "SOCKS5 password (required)")
	_ = startCmd.MarkFlagRequired("user")
	_ = startCmd.MarkFlagRequired("pass")

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop SOCKS5 listener(s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return StopCmd(cmd, con)
		},
		Example: `~~~
socks5 stop --port 1080
socks5 stop              # stop all listeners for current session
socks5 stop --all        # stop all listeners in this client
~~~`,
	}
	stopCmd.Flags().IntP("port", "p", 0, "local port to stop (0 = all for session)")
	stopCmd.Flags().String("id", "", "listener id to stop")
	stopCmd.Flags().String("session", "", "session id (default: interactive session)")
	stopCmd.Flags().BoolP("all", "a", false, "stop all SOCKS listeners in this client process")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List native SOCKS5 listeners (session-bound; use --all for global)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListCmd(cmd, con)
		},
		Example: `~~~
socks5 list              # current session (or all if no session)
socks5 list --all        # all sessions in this client
socks5 list --session <id>
~~~`,
	}
	listCmd.Flags().BoolP("all", "a", false, "list all sessions in this client process")
	listCmd.Flags().String("session", "", "filter by session id")

	cmd.AddCommand(startCmd, stopCmd, listCmd)
	return []*cobra.Command{cmd}
}

func Register(con *core.Console) {
	// Register module name only. Stream events are handled exactly once via
	// sessionRelay's AddDoneCallback (fan-out to all listeners on that session).
	con.RegisterImplantFunc(
		consts.ModuleTcpRelay,
		func(rpc clientrpc.MaliceRPCClient, sess *client.Session) (*clientpb.Task, error) {
			return nil, fmt.Errorf("use socks5 start")
		},
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) { return nil, nil },
		nil,
	)
	_ = intermediate.RegisterInternalDoneCallback(consts.ModuleTcpRelay, func(content *clientpb.TaskContext) (string, error) {
		return "", nil
	})
}

func StartCmd(cmd *cobra.Command, con *core.Console) error {
	sess := con.GetInteractive()
	if sess == nil {
		return fmt.Errorf("no interactive session")
	}
	port, _ := cmd.Flags().GetInt("port")
	bind, _ := cmd.Flags().GetString("bind")
	user, _ := cmd.Flags().GetString("user")
	pass, _ := cmd.Flags().GetString("pass")
	if user == "" || pass == "" {
		return fmt.Errorf("--user and --pass are required")
	}
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port")
	}
	if bind == "" {
		bind = "127.0.0.1"
	}

	rel := registry.getRelay(sess.SessionId)

	svc := &SocksService{
		id:        shortID(sess.SessionId, bind, port),
		sessionID: sess.SessionId,
		rpc:       con.Rpc,
		sess:      sess,
		con:       con,
		relay:     rel,
		user:      user,
		pass:      pass,
		bind:      bind,
		port:      port,
		status:    StatusStarting,
		createdAt: time.Now(),
		conns:     make(map[uint32]*localConn),
		stopCh:    make(chan struct{}),
	}

	// Bind local port first so a listen failure never leaves an orphan relay.
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bind, port))
	if err != nil {
		return fmt.Errorf("listen %s:%d: %w", bind, port, err)
	}
	svc.listener = ln

	if err := registry.addService(svc); err != nil {
		_ = ln.Close()
		return err
	}

	// Shared tcp_relay for this session (idempotent START on server).
	if err := ensureSessionRelay(con, sess, rel); err != nil {
		_ = ln.Close()
		_ = registry.removeService(svc)
		// If this was the only listener, drop client-side relay cache (no START
		// succeeded so stopRelay is unnecessary when task is nil).
		if len(registry.sessionServices(sess.SessionId)) == 0 {
			registry.clearRelay(sess.SessionId)
		}
		return err
	}
	svc.status = StatusListening

	svc.wg.Add(1)
	go svc.acceptLoop()

	con.Log.Infof("socks5 id=%s listening on %s:%d (user=%s) session=%s task=%d\n",
		svc.id, bind, port, user, shortSession(sess.SessionId), taskID(rel.task))
	return nil
}

func ensureSessionRelay(con *core.Console, sess *client.Session, rel *sessionRelay) error {
	rel.mu.Lock()
	defer rel.mu.Unlock()
	if rel.task != nil && rel.callbackInstalled {
		return nil
	}

	task, err := con.Rpc.TcpRelay(sess.Context(), &implantpb.TunnelCtrl{
		Action:   implantpb.TunnelCtrl_START,
		MaxConns: 64,
		ReadBuf:  16 * 1024,
	})
	if err != nil {
		return fmt.Errorf("start tcp_relay: %w", err)
	}
	rel.task = task
	if !rel.callbackInstalled {
		sid := sess.SessionId
		con.AddDoneCallback(task, func(ctx *clientpb.TaskContext) {
			if ctx == nil || ctx.Spite == nil {
				return
			}
			registry.routeSpite(sid, ctx.Spite)
		})
		rel.callbackInstalled = true
	}
	if rel.owners == nil {
		rel.owners = make(map[uint32]*SocksService)
	}
	if rel.pendingOpen == nil {
		rel.pendingOpen = make(map[uint32]*implantpb.TunnelOpenResult)
	}
	rel.openFailStreak = 0
	rel.lastError = ""
	return nil
}

func StopCmd(cmd *cobra.Command, con *core.Console) error {
	all, _ := cmd.Flags().GetBool("all")
	port, _ := cmd.Flags().GetInt("port")
	id, _ := cmd.Flags().GetString("id")
	sessionFlag, _ := cmd.Flags().GetString("session")

	if all {
		list := registry.list("")
		if len(list) == 0 {
			con.Log.Console("no active native socks5 listeners\n")
			return nil
		}
		// Group by session for relay STOP.
		seen := map[string]struct{}{}
		for _, svc := range list {
			stopOne(con, svc, false)
			seen[svc.sessionID] = struct{}{}
		}
		for sid := range seen {
			if len(registry.sessionServices(sid)) == 0 {
				stopRelay(con, sid)
			}
		}
		con.Log.Infof("stopped %d socks5 listener(s)\n", len(list))
		return nil
	}

	sessID := sessionFlag
	if sessID == "" {
		if s := con.GetInteractive(); s != nil {
			sessID = s.SessionId
		}
	}

	if id != "" {
		svc := registry.getByID(id)
		if svc == nil {
			return fmt.Errorf("no socks5 listener id=%s", id)
		}
		stopOne(con, svc, true)
		return nil
	}

	if sessID == "" {
		return fmt.Errorf("no session context; use --session, --id, --port with session, or --all")
	}

	if port > 0 {
		svc := registry.getByPort(sessID, port)
		if svc == nil {
			return fmt.Errorf("no socks5 on port %d for session %s", port, shortSession(sessID))
		}
		stopOne(con, svc, true)
		return nil
	}

	// No port: stop all listeners for this session.
	list := registry.sessionServices(sessID)
	if len(list) == 0 {
		return fmt.Errorf("no active socks5 for session %s", shortSession(sessID))
	}
	for _, svc := range list {
		stopOne(con, svc, false)
	}
	stopRelay(con, sessID)
	con.Log.Infof("stopped %d socks5 listener(s) for session %s\n", len(list), shortSession(sessID))
	return nil
}

func stopOne(con *core.Console, svc *SocksService, stopRelayIfLast bool) {
	svc.status = StatusStopping
	svc.Close()
	empty := registry.removeService(svc)
	con.Log.Infof("socks5 stopped id=%s %s:%d session=%s\n",
		svc.id, svc.bind, svc.port, shortSession(svc.sessionID))
	if stopRelayIfLast && empty {
		stopRelay(con, svc.sessionID)
	}
}

func stopRelay(con *core.Console, sessionID string) {
	rel := registry.getRelay(sessionID)
	rel.mu.Lock()
	task := rel.task
	rel.task = nil
	rel.callbackInstalled = false
	rel.mu.Unlock()
	registry.clearRelay(sessionID)

	if task == nil {
		return
	}
	ctx, err := sessionContext(con, sessionID)
	if err != nil {
		if con != nil && con.Log != nil {
			con.Log.Errorf("socks5 stopRelay session %s: %v\n", shortSession(sessionID), err)
		}
		return
	}
	_, _ = con.Rpc.TcpRelay(ctx, &implantpb.TunnelCtrl{Action: implantpb.TunnelCtrl_STOP})
}

// sessionContext builds outgoing gRPC metadata for the given implant session.
// Prefer a live Session from the local map so STOP/Open always target that session,
// not whatever is currently interactive.
func sessionContext(con *core.Console, sessionID string) (context.Context, error) {
	if con == nil {
		return nil, fmt.Errorf("no console")
	}
	if sess, ok := con.GetLocalSession(sessionID); ok && sess != nil {
		return sess.Context(), nil
	}
	if s := con.GetInteractive(); s != nil && s.SessionId == sessionID {
		return s.Context(), nil
	}
	// Last resort: synthesize metadata without a Session object (still correct sid).
	if sessionID == "" {
		return nil, fmt.Errorf("empty session id")
	}
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", sessionID,
		"callee", consts.CalleeCMD,
	)), nil
}

func ListCmd(cmd *cobra.Command, con *core.Console) error {
	all, _ := cmd.Flags().GetBool("all")
	sessionFlag, _ := cmd.Flags().GetString("session")

	filter := sessionFlag
	if !all && filter == "" {
		if s := con.GetInteractive(); s != nil {
			filter = s.SessionId
		}
		// No interactive session and no --session: show all (global in this process).
		if filter == "" {
			all = true
		}
	}
	if all {
		filter = ""
	}

	list := registry.list(filter)
	if len(list) == 0 {
		if filter != "" {
			con.Log.Console(fmt.Sprintf("no active native socks5 listeners for session %s\n", shortSession(filter)))
		} else {
			con.Log.Console("no active native socks5 listeners\n")
		}
		return nil
	}

	con.Log.Console(fmt.Sprintf("%-22s %-10s %-16s %-6s %-12s %-6s %-6s %-10s %s\n",
		"ID", "SESSION", "BIND", "PORT", "USER", "TASK", "CONNS", "STATUS", "LAST_ERROR"))
	for _, svc := range list {
		svc.mu.Lock()
		n := len(svc.conns)
		st := svc.status
		le := svc.lastError
		svc.mu.Unlock()
		tid := uint32(0)
		if svc.relay != nil {
			svc.relay.mu.Lock()
			tid = taskID(svc.relay.task)
			if svc.relay.lastError != "" && le == "" {
				le = svc.relay.lastError
			}
			svc.relay.mu.Unlock()
		}
		if le == "" {
			le = "-"
		}
		con.Log.Console(fmt.Sprintf("%-22s %-10s %-16s %-6d %-12s %-6d %-6d %-10s %s\n",
			svc.id, shortSession(svc.sessionID), svc.bind, svc.port, svc.user, tid, n, st, le))
	}
	return nil
}

func taskID(t *clientpb.Task) uint32 {
	if t == nil {
		return 0
	}
	return t.TaskId
}

func (s *SocksService) setError(err string) {
	s.mu.Lock()
	s.lastError = err
	if err != "" && s.status == StatusListening {
		s.status = StatusDegraded
	}
	if err == "" && s.status == StatusDegraded {
		s.status = StatusListening
	}
	s.mu.Unlock()
	if s.relay == nil {
		return
	}
	s.relay.mu.Lock()
	if err != "" {
		s.relay.openFailStreak++
		s.relay.lastError = err
		// After repeated open failures, drop cached task so the next start /
		// ensureSessionRelay performs a fresh TcpRelay START (covers half-dead streams).
		if s.relay.openFailStreak >= openFailRebuildThreshold {
			s.relay.task = nil
			s.relay.callbackInstalled = false
			s.relay.openFailStreak = 0
		}
	} else {
		s.relay.openFailStreak = 0
		s.relay.lastError = ""
	}
	s.relay.mu.Unlock()
}

func (s *SocksService) Close() {
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.mu.Lock()
	for id, c := range s.conns {
		c.closed.Store(true)
		select {
		case c.dataWait <- struct{}{}:
		default:
		}
		if c.conn != nil {
			_ = c.conn.Close()
		}
		delete(s.conns, id)
	}
	s.mu.Unlock()

	// Unregister any remaining owner bindings for this service.
	if s.relay != nil {
		ids := make([]uint32, 0)
		s.relay.mu.Lock()
		for cid, owner := range s.relay.owners {
			if owner == s {
				ids = append(ids, cid)
			}
		}
		for _, cid := range ids {
			delete(s.relay.owners, cid)
			if s.relay.pendingOpen != nil {
				delete(s.relay.pendingOpen, cid)
			}
		}
		s.relay.mu.Unlock()
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(closeWaitTimeout):
		// In-flight handlers will exit on stopCh / closed conns; do not block stop forever.
	}
}

func (s *SocksService) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				return
			}
		}
		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			s.handleClient(c)
		}(conn)
	}
}

func (s *SocksService) handleClient(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	buf := make([]byte, 258)
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	if buf[0] != 0x05 {
		return
	}
	nMethods := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:nMethods]); err != nil {
		return
	}
	hasUserPass := false
	for i := 0; i < nMethods; i++ {
		if buf[i] == 0x02 {
			hasUserPass = true
			break
		}
	}
	if !hasUserPass {
		_, _ = conn.Write([]byte{0x05, 0xFF})
		return
	}
	_, _ = conn.Write([]byte{0x05, 0x02})

	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	if buf[0] != 0x01 {
		return
	}
	ulen := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:ulen+1]); err != nil {
		return
	}
	username := string(buf[:ulen])
	plen := int(buf[ulen])
	if _, err := io.ReadFull(conn, buf[:plen]); err != nil {
		return
	}
	password := string(buf[:plen])
	if username != s.user || password != s.pass {
		_, _ = conn.Write([]byte{0x01, 0x01})
		return
	}
	_, _ = conn.Write([]byte{0x01, 0x00})

	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return
	}
	if buf[0] != 0x05 || buf[1] != 0x01 {
		s.replySocks(conn, 0x07)
		return
	}
	atyp := buf[3]
	var host string
	switch atyp {
	case 0x01:
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 0x03:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return
		}
		dlen := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:dlen]); err != nil {
			return
		}
		host = string(buf[:dlen])
	case 0x04:
		if _, err := io.ReadFull(conn, buf[:16]); err != nil {
			return
		}
		host = net.IP(buf[:16]).String()
	default:
		s.replySocks(conn, 0x08)
		return
	}
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(buf[:2])

	_ = conn.SetDeadline(time.Time{})

	// Session-wide unique conn id (shared relay).
	connID := s.relay.nextID.Add(1) - 1
	lc := &localConn{
		id:       connID,
		conn:     conn,
		openCh:   make(chan *implantpb.TunnelOpenResult, 1),
		dataWait: make(chan struct{}, 1),
	}
	s.mu.Lock()
	s.conns[connID] = lc
	s.mu.Unlock()
	// Register owner before TunnelOpen so OpenResult routes to this listener.
	if pend := s.relay.registerConn(connID, s); pend != nil {
		select {
		case lc.openCh <- pend:
		default:
		}
	}
	defer func() {
		s.mu.Lock()
		delete(s.conns, connID)
		s.mu.Unlock()
		s.relay.unregisterConn(connID)
		lc.closed.Store(true)
		select {
		case lc.dataWait <- struct{}{}:
		default:
		}
		ctx, cancel := context.WithTimeout(s.sess.Context(), 5*time.Second)
		_, _ = s.rpc.TunnelClose(ctx, &implantpb.TunnelClose{ConnId: connID, Reason: "local close"})
		cancel()
	}()

	ctx, cancel := context.WithTimeout(s.sess.Context(), 20*time.Second)
	_, err := s.rpc.TunnelOpen(ctx, &implantpb.TunnelOpen{
		ConnId:    connID,
		Host:      host,
		Port:      uint32(port),
		TimeoutMs: 10000,
	})
	cancel()
	if err != nil {
		s.setError(err.Error())
		s.replySocks(conn, 0x01)
		return
	}

	var openRes *implantpb.TunnelOpenResult
	select {
	case openRes = <-lc.openCh:
	case <-time.After(25 * time.Second):
		s.setError("TunnelOpenResult timeout")
		s.replySocks(conn, 0x01)
		return
	case <-s.stopCh:
		return
	}
	if openRes == nil || !openRes.Success {
		msg := "connect failed"
		if openRes != nil && openRes.Error != "" {
			msg = openRes.Error
		}
		s.setError(msg)
		s.replySocks(conn, 0x05)
		return
	}
	s.setError("")
	s.replySocks(conn, 0x00)

	errCh := make(chan struct{}, 2)

	go func() {
		buf := make([]byte, 16*1024)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				ctx, cancel := context.WithTimeout(s.sess.Context(), 15*time.Second)
				_, _ = s.rpc.TunnelData(ctx, &implantpb.TunnelData{
					ConnId: connID,
					Data:   append([]byte(nil), buf[:n]...),
				})
				cancel()
			}
			if err != nil {
				errCh <- struct{}{}
				return
			}
		}
	}()

	go func() {
		for {
			if lc.closed.Load() {
				errCh <- struct{}{}
				return
			}
			lc.dataMu.Lock()
			var data []byte
			if len(lc.dataQ) > 0 {
				data = lc.dataQ[0]
				lc.dataQ[0] = nil
				lc.dataQ = lc.dataQ[1:]
			}
			lc.dataMu.Unlock()
			if data == nil {
				select {
				case <-lc.dataWait:
					continue
				case <-s.stopCh:
					errCh <- struct{}{}
					return
				}
			}
			if _, err := conn.Write(data); err != nil {
				errCh <- struct{}{}
				return
			}
		}
	}()

	<-errCh
}

func (s *SocksService) replySocks(conn net.Conn, rep byte) {
	_, _ = conn.Write([]byte{0x05, rep, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
}

func (s *SocksService) handleSpite(spite *implantpb.Spite) {
	if spite == nil {
		return
	}
	if r := spite.GetTunnelOpenResult(); r != nil {
		s.mu.Lock()
		lc := s.conns[r.ConnId]
		s.mu.Unlock()
		if lc == nil {
			// Owner registered but local conn map race: buffer on session relay.
			if s.relay != nil {
				s.relay.mu.Lock()
				if s.relay.pendingOpen == nil {
					s.relay.pendingOpen = make(map[uint32]*implantpb.TunnelOpenResult)
				}
				if len(s.relay.pendingOpen) < 1024 {
					s.relay.pendingOpen[r.ConnId] = r
				}
				s.relay.mu.Unlock()
			}
			return
		}
		select {
		case lc.openCh <- r:
		default:
		}
		return
	}
	if d := spite.GetTunnelData(); d != nil {
		s.mu.Lock()
		lc := s.conns[d.ConnId]
		s.mu.Unlock()
		if lc == nil || lc.closed.Load() {
			return
		}
		if len(d.Data) > 0 {
			chunk := append([]byte(nil), d.Data...)
			lc.dataMu.Lock()
			if len(lc.dataQ) >= maxDataQChunks {
				lc.dataMu.Unlock()
				// Backpressure: close rather than drop bytes (TLS integrity).
				lc.closed.Store(true)
				select {
				case lc.dataWait <- struct{}{}:
				default:
				}
				if lc.conn != nil {
					_ = lc.conn.Close()
				}
				return
			}
			lc.dataQ = append(lc.dataQ, chunk)
			lc.dataMu.Unlock()
			select {
			case lc.dataWait <- struct{}{}:
			default:
			}
		}
		if d.Fin {
			lc.closed.Store(true)
			select {
			case lc.dataWait <- struct{}{}:
			default:
			}
			if lc.conn != nil {
				_ = lc.conn.Close()
			}
		}
		return
	}
	if c := spite.GetTunnelClose(); c != nil {
		s.mu.Lock()
		lc := s.conns[c.ConnId]
		s.mu.Unlock()
		if lc != nil {
			lc.closed.Store(true)
			select {
			case lc.dataWait <- struct{}{}:
			default:
			}
			if lc.conn != nil {
				_ = lc.conn.Close()
			}
		}
	}
}

