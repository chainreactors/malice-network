//go:build mockimplant

package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	iomtypes "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	implantparser "github.com/chainreactors/malice-network/server/internal/parser"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
	"github.com/chainreactors/malice-network/server/testsupport"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func TestMockImplantParserListenerHTTPE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-http-listener-%d", time.Now().UnixNano())
	pipeline := h.NewHTTPPipeline(t, "mock-parser-http-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetHttp().Port = uint32(reserveTCPPort(t))
	pipeline.GetHttp().Params = (&implanttypes.PipelineParams{
		Headers:    map[string][]string{"X-Test": {"integration"}},
		BodyPrefix: "prefix:",
		BodySuffix: ":suffix",
	}).String()

	startInProcessListener(t, h, listenerName)
	startPipeline(t, pipeline)

	implant := newHTTPMockImplant(t, pipeline)

	respSpites, resp, err := implant.Exchange(&implantpb.Spite{
		Name: iomtypes.MsgRegister.String(),
		Body: &implantpb.Spite_Register{
			Register: &implantpb.Register{
				Name: "http-mockimplant",
				Timer: &implantpb.Timer{
					Expression: "* * * * *",
				},
				Sysinfo: &implantpb.SysInfo{
					Workdir: `C:\http\work`,
					Os: &implantpb.Os{
						Name:     "windows",
						Arch:     "amd64",
						Hostname: "http-mock-host",
					},
					Process: &implantpb.Process{
						Name: "http-mock.exe",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("http register failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Test"); got != "integration" {
		t.Fatalf("register header X-Test = %q, want integration", got)
	}
	if len(respSpites) != 0 {
		t.Fatalf("register response spites = %d, want 0", len(respSpites))
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, err := core.Sessions.Get(implant.SessionID)
		return err == nil
	}, "http session registration through parser/listener")

	runtimeSession, err := core.Sessions.Get(implant.SessionID)
	if err != nil {
		t.Fatalf("core.Sessions.Get failed: %v", err)
	}
	if runtimeSession.Name != "http-mockimplant" {
		t.Fatalf("runtime session name = %q, want http-mockimplant", runtimeSession.Name)
	}
	if runtimeSession.PipelineID != pipeline.Name {
		t.Fatalf("runtime session pipeline = %q, want %q", runtimeSession.PipelineID, pipeline.Name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	rpc := clientrpc.NewMaliceRPCClient(conn)
	sessionCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", implant.SessionID,
		"callee", consts.CalleeCMD,
	))

	task, err := rpc.Pwd(sessionCtx, &implantpb.Request{Name: consts.ModulePwd})
	if err != nil {
		t.Fatalf("Pwd failed: %v", err)
	}
	if task == nil || task.TaskId == 0 {
		t.Fatalf("Pwd task = %#v, want valid task", task)
	}

	request, requestResp, err := waitForHTTPModuleRequest(implant, consts.ModulePwd, 5*time.Second)
	if err != nil {
		t.Fatalf("wait for http task request failed: %v", err)
	}
	if requestResp.StatusCode != http.StatusOK {
		t.Fatalf("poll status = %d, want 200", requestResp.StatusCode)
	}
	if got := requestResp.Header.Get("X-Test"); got != "integration" {
		t.Fatalf("poll header X-Test = %q, want integration", got)
	}
	if request.GetTaskId() != task.TaskId {
		t.Fatalf("pwd task id = %d, want %d", request.GetTaskId(), task.TaskId)
	}
	if request.GetRequest().GetName() != consts.ModulePwd {
		t.Fatalf("pwd request name = %q, want %q", request.GetRequest().GetName(), consts.ModulePwd)
	}

	responseSpites, responseResp, err := implant.Exchange(&implantpb.Spite{
		Name:   consts.ModulePwd,
		TaskId: task.TaskId,
		Body: &implantpb.Spite_Response{
			Response: &implantpb.Response{
				Output: `C:\http\work`,
			},
		},
	})
	if err != nil {
		t.Fatalf("http task response failed: %v", err)
	}
	if responseResp.StatusCode != http.StatusOK {
		t.Fatalf("response status = %d, want 200", responseResp.StatusCode)
	}
	if len(responseSpites) != 0 {
		t.Fatalf("task response spites = %d, want 0", len(responseSpites))
	}

	content := waitTaskFinish(t, rpc, implant.SessionID, task.TaskId)
	if got := content.GetSpite().GetResponse().GetOutput(); got != `C:\http\work` {
		t.Fatalf("pwd output = %q, want %q", got, `C:\http\work`)
	}
}

func TestMockImplantParserListenerHTTPRejectsMalformedPacket(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-http-bad-listener-%d", time.Now().UnixNano())
	pipeline := h.NewHTTPPipeline(t, "mock-parser-http-bad-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetHttp().Port = uint32(reserveTCPPort(t))
	pipeline.GetHttp().Params = (&implanttypes.PipelineParams{
		ErrorPage: "<html>bad-request</html>",
	}).String()

	startInProcessListener(t, h, listenerName)
	startPipeline(t, pipeline)

	implant := newHTTPMockImplant(t, pipeline)

	malformed := make([]byte, 9)
	malformed[0] = 0x00
	binary.LittleEndian.PutUint32(malformed[1:5], implant.RawID)
	binary.LittleEndian.PutUint32(malformed[5:9], 0)

	respSpites, resp, err := implant.ExchangePlain(malformed)
	if err != nil {
		t.Fatalf("http malformed request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("malformed status = %d, want 200", resp.StatusCode)
	}
	if len(respSpites) != 0 {
		t.Fatalf("malformed response spites = %d, want 0", len(respSpites))
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return core.Connections.Get(implant.SessionID) == nil
	}, "http runtime connection cleanup for malformed packet")

	if _, err := core.Sessions.Get(implant.SessionID); err == nil {
		t.Fatalf("session %s should not be registered from malformed http packet", implant.SessionID)
	}
	if session, err := h.GetSession(implant.SessionID); err == nil && session != nil {
		t.Fatalf("database session = %#v, want nil for malformed http packet", session)
	}
	if conn := core.Connections.Get(implant.SessionID); conn != nil {
		t.Fatalf("runtime connection should be removed after malformed http packet, got %#v", conn)
	}
}

func TestMockImplantParserListenerHTTPRejectsWrongEncryptionKey(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-http-crypt-listener-%d", time.Now().UnixNano())
	pipeline := h.NewHTTPPipeline(t, "mock-parser-http-crypt-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetHttp().Port = uint32(reserveTCPPort(t))

	startInProcessListener(t, h, listenerName)
	startPipeline(t, pipeline)

	wrongPipeline := proto.Clone(pipeline).(*clientpb.Pipeline)
	wrongPipeline.Encryption = []*clientpb.Encryption{
		{
			Type: pipeline.GetEncryption()[0].GetType(),
			Key:  "wrong-http-secret",
		},
	}
	implant := newHTTPMockImplant(t, wrongPipeline)

	respSpites, resp, err := implant.Exchange(&implantpb.Spite{
		Name: iomtypes.MsgRegister.String(),
		Body: &implantpb.Spite_Register{
			Register: &implantpb.Register{
				Name: "wrong-http-crypto-implant",
				Timer: &implantpb.Timer{
					Expression: "* * * * *",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("http wrong-encryption request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("wrong-encryption status = %d, want 200", resp.StatusCode)
	}
	if len(respSpites) != 0 {
		t.Fatalf("wrong-encryption response spites = %d, want 0", len(respSpites))
	}

	if _, err := core.Sessions.Get(implant.SessionID); err == nil {
		t.Fatalf("session %s should not be registered with wrong http encryption", implant.SessionID)
	}
	if session, err := h.GetSession(implant.SessionID); err == nil && session != nil {
		t.Fatalf("database session = %#v, want nil for wrong http encryption", session)
	}
	if conn := core.Connections.Get(implant.SessionID); conn != nil {
		t.Fatalf("runtime connection should not exist for wrong http encryption, got %#v", conn)
	}
}

func TestMockImplantParserListenerHTTPTLSE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-http-tls-listener-%d", time.Now().UnixNano())
	pipeline := h.NewHTTPPipeline(t, "mock-parser-http-tls-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetHttp().Port = uint32(reserveTCPPort(t))
	pipeline.GetHttp().Params = (&implanttypes.PipelineParams{
		Headers: map[string][]string{"X-TLS": {"true"}},
	}).String()
	enableSelfSignedTLS(t, pipeline)

	startInProcessListener(t, h, listenerName)
	startPipeline(t, pipeline)

	implant := newHTTPMockImplant(t, pipeline)

	respSpites, resp, err := implant.Exchange(&implantpb.Spite{
		Name: iomtypes.MsgRegister.String(),
		Body: &implantpb.Spite_Register{
			Register: &implantpb.Register{
				Name: "http-tls-mockimplant",
				Timer: &implantpb.Timer{
					Expression: "* * * * *",
				},
				Sysinfo: &implantpb.SysInfo{
					Workdir: `C:\http\tls`,
					Os: &implantpb.Os{
						Name:     "windows",
						Arch:     "amd64",
						Hostname: "http-tls-host",
					},
					Process: &implantpb.Process{
						Name: "http-tls.exe",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("https register failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("https register status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("X-TLS"); got != "true" {
		t.Fatalf("https register header X-TLS = %q, want true", got)
	}
	if len(respSpites) != 0 {
		t.Fatalf("https register response spites = %d, want 0", len(respSpites))
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, err := core.Sessions.Get(implant.SessionID)
		return err == nil
	}, "https session registration through parser/listener")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	rpc := clientrpc.NewMaliceRPCClient(conn)
	sessionCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"session_id", implant.SessionID,
		"callee", consts.CalleeCMD,
	))

	task, err := rpc.Pwd(sessionCtx, &implantpb.Request{Name: consts.ModulePwd})
	if err != nil {
		t.Fatalf("Pwd failed: %v", err)
	}
	if task == nil || task.TaskId == 0 {
		t.Fatalf("Pwd task = %#v, want valid task", task)
	}

	request, requestResp, err := waitForHTTPModuleRequest(implant, consts.ModulePwd, 5*time.Second)
	if err != nil {
		t.Fatalf("wait for https task request failed: %v", err)
	}
	if requestResp.StatusCode != http.StatusOK {
		t.Fatalf("https poll status = %d, want 200", requestResp.StatusCode)
	}
	if request.GetTaskId() != task.TaskId {
		t.Fatalf("https pwd task id = %d, want %d", request.GetTaskId(), task.TaskId)
	}

	responseSpites, responseResp, err := implant.Exchange(&implantpb.Spite{
		Name:   consts.ModulePwd,
		TaskId: task.TaskId,
		Body: &implantpb.Spite_Response{
			Response: &implantpb.Response{
				Output: `C:\http\tls`,
			},
		},
	})
	if err != nil {
		t.Fatalf("https task response failed: %v", err)
	}
	if responseResp.StatusCode != http.StatusOK {
		t.Fatalf("https response status = %d, want 200", responseResp.StatusCode)
	}
	if len(responseSpites) != 0 {
		t.Fatalf("https response spites = %d, want 0", len(responseSpites))
	}

	content := waitTaskFinish(t, rpc, implant.SessionID, task.TaskId)
	if got := content.GetSpite().GetResponse().GetOutput(); got != `C:\http\tls` {
		t.Fatalf("https pwd output = %q, want %q", got, `C:\http\tls`)
	}
}

type httpMockImplant struct {
	t           testing.TB
	RawID       uint32
	SessionID   string
	parser      *implantparser.MessageParser
	endpoint    string
	encryptions []*clientpb.Encryption
	bodyPrefix  []byte
	bodySuffix  []byte
	client      *http.Client
}

func newHTTPMockImplant(t testing.TB, pipeline *clientpb.Pipeline) *httpMockImplant {
	t.Helper()

	if pipeline == nil || pipeline.GetHttp() == nil {
		t.Fatal("http pipeline is nil")
	}

	params, err := implanttypes.UnmarshalPipelineParams(pipeline.GetHttp().GetParams())
	if err != nil {
		t.Fatalf("unmarshal http params failed: %v", err)
	}

	parser, err := implantparser.NewParser(pipeline.GetParser())
	if err != nil {
		t.Fatalf("build parser failed: %v", err)
	}

	scheme := "http"
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if pipeline.GetTls().GetEnable() {
		clientTLSConfig, err := newClientTLSConfig(pipeline.GetTls())
		if err != nil {
			t.Fatalf("build https client tls config failed: %v", err)
		}
		transport.TLSClientConfig = clientTLSConfig
		scheme = "https"
	}

	rawID := uint32(time.Now().UnixNano())
	return &httpMockImplant{
		t:           t,
		RawID:       rawID,
		SessionID:   hash.Md5Hash(encoders.Uint32ToBytes(rawID)),
		parser:      parser,
		endpoint:    fmt.Sprintf("%s://%s:%d/", scheme, pipeline.GetHttp().GetHost(), pipeline.GetHttp().GetPort()),
		encryptions: pipeline.GetEncryption(),
		bodyPrefix:  []byte(params.BodyPrefix),
		bodySuffix:  []byte(params.BodySuffix),
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
	}
}

func (i *httpMockImplant) Exchange(spite *implantpb.Spite) ([]*implantpb.Spite, *http.Response, error) {
	if i == nil {
		return nil, nil, errors.New("http mock implant is nil")
	}
	if spite == nil {
		return nil, nil, errors.New("spite is nil")
	}

	packet, err := i.parser.Marshal(iomtypes.BuildOneSpites(spite), i.RawID)
	if err != nil {
		return nil, nil, err
	}
	return i.ExchangePlain(packet)
}

func (i *httpMockImplant) ExchangePlain(packet []byte) ([]*implantpb.Spite, *http.Response, error) {
	if i == nil {
		return nil, nil, errors.New("http mock implant is nil")
	}
	if len(packet) == 0 {
		return nil, nil, errors.New("packet is empty")
	}

	cryptos, err := cryptosForRequest(i.encryptions)
	if err != nil {
		return nil, nil, err
	}

	requestBody, err := cryptostream.Encrypt(cryptos[0], packet)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, i.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := i.client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, resp, err
	}

	body, err = trimHTTPBodyEnvelope(body, i.bodyPrefix, i.bodySuffix)
	if err != nil {
		return nil, resp, err
	}
	if len(body) == 0 {
		return nil, resp, nil
	}

	plainResponse, err := cryptostream.Decrypt(cryptos[0], body)
	if err != nil {
		return nil, resp, err
	}

	responseConn := cryptostream.WrapReadWriteCloser(bytes.NewReader(plainResponse), io.Discard, nil)
	rawID, spites, err := i.parser.ReadPacket(responseConn)
	if err != nil {
		return nil, resp, err
	}
	if rawID != i.RawID {
		return nil, resp, fmt.Errorf("http response raw id = %d, want %d", rawID, i.RawID)
	}
	if spites == nil {
		return nil, resp, nil
	}
	return spites.Spites, resp, nil
}

func findSpiteByName(spites []*implantpb.Spite, name string) *implantpb.Spite {
	for _, spite := range spites {
		if spite == nil {
			continue
		}
		if spite.GetName() == name {
			return spite
		}
	}
	return nil
}

func waitForHTTPModuleRequest(implant *httpMockImplant, module string, timeout time.Duration) (*implantpb.Spite, *http.Response, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		spites, resp, err := implant.Exchange(iomtypes.BuildPingSpite())
		if err != nil {
			return nil, nil, err
		}
		if spite := findSpiteByName(spites, module); spite != nil {
			return spite, resp, nil
		}
	}
	return nil, nil, fmt.Errorf("timed out waiting for http request %s", module)
}

func trimHTTPBodyEnvelope(body, prefix, suffix []byte) ([]byte, error) {
	if len(body) == 0 {
		return nil, nil
	}
	if len(prefix) > 0 {
		if !bytes.HasPrefix(body, prefix) {
			return nil, fmt.Errorf("response body missing prefix %q", string(prefix))
		}
		body = body[len(prefix):]
	}
	if len(suffix) > 0 {
		if !bytes.HasSuffix(body, suffix) {
			return nil, fmt.Errorf("response body missing suffix %q", string(suffix))
		}
		body = body[:len(body)-len(suffix)]
	}
	return body, nil
}

func cryptosForRequest(encryptions []*clientpb.Encryption) ([]cryptostream.Cryptor, error) {
	cryptos, err := configs.NewCrypto(encryptions)
	if err != nil {
		return nil, err
	}
	if len(cryptos) == 0 {
		return nil, errors.New("pipeline has no encryption configured")
	}
	return cryptos, nil
}
