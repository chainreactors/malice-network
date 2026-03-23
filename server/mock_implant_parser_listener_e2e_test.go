//go:build mockimplant

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	iomtypes "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	implantparser "github.com/chainreactors/malice-network/server/internal/parser"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
	"github.com/chainreactors/malice-network/server/listener"
	serverrpc "github.com/chainreactors/malice-network/server/rpc"
	"github.com/chainreactors/malice-network/server/testsupport"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

func TestMockImplantParserListenerTCPE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(reserveTCPPort(t))

	startInProcessListener(t, h, listenerName)
	startPipeline(t, pipeline)

	implant := newTCPMockImplant(t, pipeline)

	register := &implantpb.Register{
		Name: "tcp-mockimplant",
		Timer: &implantpb.Timer{
			Expression: "* * * * *",
		},
		Sysinfo: &implantpb.SysInfo{
			Workdir: `C:\integration\work`,
			Os: &implantpb.Os{
				Name:     "windows",
				Arch:     "amd64",
				Hostname: "tcp-mock-host",
			},
			Process: &implantpb.Process{
				Name: "tcp-mock.exe",
			},
		},
	}
	if err := implant.Send(&implantpb.Spite{
		Name: iomtypes.MsgRegister.String(),
		Body: &implantpb.Spite_Register{Register: register},
	}); err != nil {
		t.Fatalf("send register packet failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, err := core.Sessions.Get(implant.SessionID)
		return err == nil
	}, "runtime session registration through parser/listener")

	runtimeSession, err := core.Sessions.Get(implant.SessionID)
	if err != nil {
		t.Fatalf("core.Sessions.Get failed: %v", err)
	}
	if runtimeSession.Name != "tcp-mockimplant" {
		t.Fatalf("runtime session name = %q, want tcp-mockimplant", runtimeSession.Name)
	}
	if runtimeSession.PipelineID != pipeline.Name {
		t.Fatalf("runtime session pipeline = %q, want %q", runtimeSession.PipelineID, pipeline.Name)
	}

	session, err := h.GetSession(implant.SessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session.GetPipelineId() != pipeline.Name {
		t.Fatalf("session pipeline = %q, want %q", session.GetPipelineId(), pipeline.Name)
	}
	if got := session.GetWorkdir(); got != `C:\integration\work` {
		t.Fatalf("session workdir = %q, want %q", got, `C:\integration\work`)
	}
	if got := strings.ToLower(session.GetOs().GetHostname()); got != "tcp-mock-host" {
		t.Fatalf("session hostname = %q, want tcp-mock-host", got)
	}

	drainServerPackets(t, implant, 200*time.Millisecond)

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

	request := waitForModuleRequest(t, implant, consts.ModulePwd, 5*time.Second)
	if request.GetTaskId() != task.TaskId {
		t.Fatalf("pwd task id = %d, want %d", request.GetTaskId(), task.TaskId)
	}
	if request.GetRequest().GetName() != consts.ModulePwd {
		t.Fatalf("pwd request name = %q, want %q", request.GetRequest().GetName(), consts.ModulePwd)
	}

	if err := implant.Send(&implantpb.Spite{
		Name:   consts.ModulePwd,
		TaskId: task.TaskId,
		Body: &implantpb.Spite_Response{
			Response: &implantpb.Response{
				Output: `C:\integration\work`,
			},
		},
	}); err != nil {
		t.Fatalf("send pwd response failed: %v", err)
	}

	content := waitTaskFinish(t, rpc, implant.SessionID, task.TaskId)
	if got := content.GetSpite().GetResponse().GetOutput(); got != `C:\integration\work` {
		t.Fatalf("pwd output = %q, want %q", got, `C:\integration\work`)
	}
}

func TestMockImplantParserListenerTCPAESE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-aes-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-aes-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(reserveTCPPort(t))
	pipeline.Encryption = []*clientpb.Encryption{
		{
			Type: consts.CryptorAES,
			Key:  "integration-secret-aes",
		},
	}

	startInProcessListener(t, h, listenerName)
	startPipeline(t, pipeline)

	implant := newTCPMockImplant(t, pipeline)

	if err := implant.Send(&implantpb.Spite{
		Name: iomtypes.MsgRegister.String(),
		Body: &implantpb.Spite_Register{
			Register: &implantpb.Register{
				Name: "tcp-aes-mockimplant",
				Timer: &implantpb.Timer{
					Expression: "* * * * *",
				},
				Sysinfo: &implantpb.SysInfo{
					Workdir: `C:\integration\aes`,
					Os: &implantpb.Os{
						Name:     "windows",
						Arch:     "amd64",
						Hostname: "tcp-aes-host",
					},
					Process: &implantpb.Process{
						Name: "tcp-aes.exe",
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("send aes register packet failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, err := core.Sessions.Get(implant.SessionID)
		return err == nil
	}, "runtime session registration through aes tcp parser/listener")

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

	request := waitForModuleRequest(t, implant, consts.ModulePwd, 5*time.Second)
	if request.GetTaskId() != task.TaskId {
		t.Fatalf("aes pwd task id = %d, want %d", request.GetTaskId(), task.TaskId)
	}

	if err := implant.Send(&implantpb.Spite{
		Name:   consts.ModulePwd,
		TaskId: task.TaskId,
		Body: &implantpb.Spite_Response{
			Response: &implantpb.Response{
				Output: `C:\integration\aes`,
			},
		},
	}); err != nil {
		t.Fatalf("send aes pwd response failed: %v", err)
	}

	content := waitTaskFinish(t, rpc, implant.SessionID, task.TaskId)
	if got := content.GetSpite().GetResponse().GetOutput(); got != `C:\integration\aes` {
		t.Fatalf("aes pwd output = %q, want %q", got, `C:\integration\aes`)
	}
}

func TestMockImplantParserListenerTCPTLSE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-tls-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-tls-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(reserveTCPPort(t))
	enableSelfSignedTLS(t, pipeline)

	startInProcessListener(t, h, listenerName)
	startPipeline(t, pipeline)

	implant := newTCPMockImplant(t, pipeline)

	if err := implant.Send(&implantpb.Spite{
		Name: iomtypes.MsgRegister.String(),
		Body: &implantpb.Spite_Register{
			Register: &implantpb.Register{
				Name: "tcp-tls-mockimplant",
				Timer: &implantpb.Timer{
					Expression: "* * * * *",
				},
				Sysinfo: &implantpb.SysInfo{
					Workdir: `C:\integration\tls`,
					Os: &implantpb.Os{
						Name:     "windows",
						Arch:     "amd64",
						Hostname: "tcp-tls-host",
					},
					Process: &implantpb.Process{
						Name: "tcp-tls.exe",
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("send tls register packet failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, err := core.Sessions.Get(implant.SessionID)
		return err == nil
	}, "runtime session registration through tls tcp parser/listener")

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

	request := waitForModuleRequest(t, implant, consts.ModulePwd, 5*time.Second)
	if request.GetTaskId() != task.TaskId {
		t.Fatalf("tls pwd task id = %d, want %d", request.GetTaskId(), task.TaskId)
	}

	if err := implant.Send(&implantpb.Spite{
		Name:   consts.ModulePwd,
		TaskId: task.TaskId,
		Body: &implantpb.Spite_Response{
			Response: &implantpb.Response{
				Output: `C:\integration\tls`,
			},
		},
	}); err != nil {
		t.Fatalf("send tls pwd response failed: %v", err)
	}

	content := waitTaskFinish(t, rpc, implant.SessionID, task.TaskId)
	if got := content.GetSpite().GetResponse().GetOutput(); got != `C:\integration\tls` {
		t.Fatalf("tls pwd output = %q, want %q", got, `C:\integration\tls`)
	}
}

func TestMockImplantParserListenerRejectsMalformedTCPPacket(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-bad-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-bad-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(reserveTCPPort(t))

	startInProcessListener(t, h, listenerName)
	startPipeline(t, pipeline)

	implant := newTCPMockImplant(t, pipeline)

	malformed := make([]byte, 9)
	malformed[0] = 0x00
	binary.LittleEndian.PutUint32(malformed[1:5], implant.RawID)
	binary.LittleEndian.PutUint32(malformed[5:9], 0)

	if err := implant.WritePlain(malformed); err != nil {
		t.Fatalf("send malformed packet failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return core.Connections.Get(implant.SessionID) == nil
	}, "runtime connection cleanup for malformed packet")

	if _, err := core.Sessions.Get(implant.SessionID); err == nil {
		t.Fatalf("session %s should not be registered from malformed packet", implant.SessionID)
	}
	if session, err := h.GetSession(implant.SessionID); err == nil && session != nil {
		t.Fatalf("database session = %#v, want nil for malformed packet", session)
	}
	if conn := core.Connections.Get(implant.SessionID); conn != nil {
		t.Fatalf("runtime connection should be removed after malformed packet, got %#v", conn)
	}
}

func TestMockImplantParserListenerRejectsWrongEncryptionKey(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-crypt-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-crypt-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(reserveTCPPort(t))

	startInProcessListener(t, h, listenerName)
	startPipeline(t, pipeline)

	wrongPipeline := proto.Clone(pipeline).(*clientpb.Pipeline)
	wrongPipeline.Encryption = []*clientpb.Encryption{
		{
			Type: pipeline.GetEncryption()[0].GetType(),
			Key:  "wrong-integration-secret",
		},
	}
	implant := newTCPMockImplant(t, wrongPipeline)

	if err := implant.Send(&implantpb.Spite{
		Name: iomtypes.MsgRegister.String(),
		Body: &implantpb.Spite_Register{
			Register: &implantpb.Register{
				Name: "wrong-crypto-implant",
				Timer: &implantpb.Timer{
					Expression: "* * * * *",
				},
			},
		},
	}); err != nil {
		t.Fatalf("send wrong-encryption register packet failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return core.Connections.Get(implant.SessionID) == nil
	}, "runtime connection cleanup for wrong encryption")

	if _, err := core.Sessions.Get(implant.SessionID); err == nil {
		t.Fatalf("session %s should not be registered with wrong encryption", implant.SessionID)
	}
	if session, err := h.GetSession(implant.SessionID); err == nil && session != nil {
		t.Fatalf("database session = %#v, want nil for wrong encryption", session)
	}
	if conn := core.Connections.Get(implant.SessionID); conn != nil {
		t.Fatalf("runtime connection should be removed after wrong encryption, got %#v", conn)
	}
}

func TestMockImplantParserListenerRejectsInvalidPayloadTerminator(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-payload-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-payload-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(reserveTCPPort(t))

	startInProcessListener(t, h, listenerName)
	startPipeline(t, pipeline)

	implant := newTCPMockImplant(t, pipeline)

	packet := make([]byte, 11)
	packet[0] = 0xd1
	binary.LittleEndian.PutUint32(packet[1:5], implant.RawID)
	binary.LittleEndian.PutUint32(packet[5:9], 1)
	packet[9] = 0x41
	packet[10] = 0x42

	if err := implant.WritePlain(packet); err != nil {
		t.Fatalf("send invalid payload packet failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return core.Connections.Get(implant.SessionID) == nil
	}, "runtime connection cleanup for invalid payload terminator")

	if _, err := core.Sessions.Get(implant.SessionID); err == nil {
		t.Fatalf("session %s should not be registered from invalid payload", implant.SessionID)
	}
	if session, err := h.GetSession(implant.SessionID); err == nil && session != nil {
		t.Fatalf("database session = %#v, want nil for invalid payload", session)
	}
	if conn := core.Connections.Get(implant.SessionID); conn != nil {
		t.Fatalf("runtime connection should be removed after invalid payload, got %#v", conn)
	}
}

func startInProcessListener(t testing.TB, h *testsupport.ControlPlaneHarness, listenerName string) {
	t.Helper()

	authConfig := h.NewListenerClientConfig(t, listenerName)
	authBytes, err := yaml.Marshal(authConfig)
	if err != nil {
		t.Fatalf("marshal listener auth: %v", err)
	}

	authPath, err := h.WriteTempFile(listenerName+".auth", authBytes)
	if err != nil {
		t.Fatalf("write listener auth: %v", err)
	}

	cfg := &configs.ListenerConfig{
		Enable: true,
		Name:   listenerName,
		Auth:   authPath,
		IP:     "127.0.0.1",
	}
	if err := listener.NewListener(authConfig, cfg, true); err != nil {
		t.Fatalf("start listener %s failed: %v", listenerName, err)
	}

	t.Cleanup(func() {
		if listener.Listener != nil {
			_ = listener.Listener.Close()
		}
	})
}

func startPipeline(t testing.TB, pipeline *clientpb.Pipeline) {
	t.Helper()

	if pipeline == nil {
		t.Fatal("pipeline is nil")
	}

	if _, err := (&serverrpc.Server{}).RegisterPipeline(context.Background(), pipeline); err != nil {
		t.Fatalf("register pipeline %s failed: %v", pipeline.Name, err)
	}
	if _, err := (&serverrpc.Server{}).StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       pipeline.Name,
		ListenerId: pipeline.ListenerId,
		Pipeline:   pipeline,
	}); err != nil {
		t.Fatalf("start pipeline %s failed: %v", pipeline.Name, err)
	}

	t.Cleanup(func() {
		_, _ = (&serverrpc.Server{}).StopPipeline(context.Background(), &clientpb.CtrlPipeline{
			Name:       pipeline.Name,
			ListenerId: pipeline.ListenerId,
			Pipeline:   pipeline,
		})
	})
}

type tcpMockImplant struct {
	t         testing.TB
	RawID     uint32
	SessionID string
	parser    *implantparser.MessageParser
	rawConn   net.Conn
	crypto    *cryptostream.CryptoConn
	pending   []*implantpb.Spite
}

func newTCPMockImplant(t testing.TB, pipeline *clientpb.Pipeline) *tcpMockImplant {
	t.Helper()

	if pipeline == nil || pipeline.GetTcp() == nil {
		t.Fatal("tcp pipeline is nil")
	}

	cryptos, err := configs.NewCrypto(pipeline.GetEncryption())
	if err != nil {
		t.Fatalf("build pipeline cryptor failed: %v", err)
	}
	if len(cryptos) == 0 {
		t.Fatal("pipeline has no encryption configured")
	}

	parser, err := implantparser.NewParser(pipeline.GetParser())
	if err != nil {
		t.Fatalf("build parser failed: %v", err)
	}

	address := fmt.Sprintf("%s:%d", pipeline.GetTcp().GetHost(), pipeline.GetTcp().GetPort())
	rawConn, err := dialPipelineConn(address, pipeline.GetTls())
	if err != nil {
		t.Fatalf("dial pipeline %s failed: %v", address, err)
	}

	rawID := uint32(time.Now().UnixNano())
	implant := &tcpMockImplant{
		t:         t,
		RawID:     rawID,
		SessionID: hash.Md5Hash(encoders.Uint32ToBytes(rawID)),
		parser:    parser,
		rawConn:   rawConn,
		crypto:    cryptostream.NewCryptoConn(rawConn, cryptos[0]),
	}

	t.Cleanup(func() {
		_ = implant.rawConn.Close()
	})

	return implant
}

func dialPipelineConn(address string, tlsConfigPB *clientpb.TLS) (net.Conn, error) {
	if tlsConfigPB == nil || !tlsConfigPB.GetEnable() {
		return net.DialTimeout("tcp", address, 5*time.Second)
	}

	clientTLSConfig, err := newClientTLSConfig(tlsConfigPB)
	if err != nil {
		return nil, err
	}
	return tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", address, clientTLSConfig)
}

func (i *tcpMockImplant) Send(spite *implantpb.Spite) error {
	if i == nil {
		return errors.New("tcp mock implant is nil")
	}
	if spite == nil {
		return errors.New("spite is nil")
	}
	return i.parser.WritePacket(i.crypto, iomtypes.BuildOneSpites(spite), i.RawID)
}

func (i *tcpMockImplant) WritePlain(data []byte) error {
	if i == nil {
		return errors.New("tcp mock implant is nil")
	}
	if len(data) == 0 {
		return errors.New("data is empty")
	}
	n, err := i.crypto.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("short write: got %d, want %d", n, len(data))
	}
	return nil
}

func (i *tcpMockImplant) Read(timeout time.Duration) (*implantpb.Spite, error) {
	if i == nil {
		return nil, errors.New("tcp mock implant is nil")
	}
	if len(i.pending) > 0 {
		spite := i.pending[0]
		i.pending = i.pending[1:]
		return spite, nil
	}
	if err := i.rawConn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	defer func() {
		_ = i.rawConn.SetReadDeadline(time.Time{})
	}()

	rawID, spites, err := i.parser.ReadPacket(i.crypto)
	if err != nil {
		return nil, err
	}
	if rawID != i.RawID {
		return nil, fmt.Errorf("server packet raw id = %d, want %d", rawID, i.RawID)
	}
	if spites == nil || len(spites.Spites) == 0 {
		return nil, errors.New("server packet contained no spites")
	}
	if len(spites.Spites) > 1 {
		i.pending = append(i.pending, spites.Spites[1:]...)
	}
	return spites.Spites[0], nil
}

func waitForModuleRequest(t testing.TB, implant *tcpMockImplant, module string, timeout time.Duration) *implantpb.Spite {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := implant.Send(iomtypes.BuildPingSpite()); err != nil {
			t.Fatalf("send ping trigger failed: %v", err)
		}

		spite, err := implant.Read(1500 * time.Millisecond)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			t.Fatalf("read server packet failed: %v", err)
		}

		if spite.GetInit() != nil || spite.GetPing() != nil {
			continue
		}
		if spite.GetName() == module {
			return spite
		}
		t.Fatalf("server request = %q, want %q", spite.GetName(), module)
	}

	t.Fatalf("timed out waiting for server request %s", module)
	return nil
}

func drainServerPackets(t testing.TB, implant *tcpMockImplant, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		spite, err := implant.Read(50 * time.Millisecond)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return
			}
			t.Fatalf("drain server packet failed: %v", err)
		}
		if spite == nil {
			return
		}
	}
}

func reserveTCPPort(t testing.TB) uint16 {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve tcp port failed: %v", err)
	}
	defer ln.Close()

	return uint16(ln.Addr().(*net.TCPAddr).Port)
}

func enableSelfSignedTLS(t testing.TB, pipeline *clientpb.Pipeline) {
	t.Helper()

	if pipeline == nil {
		t.Fatal("pipeline is nil")
	}

	caCertPEM, caKeyPEM, err := certs.GenerateCACert("test-pipeline", nil)
	if err != nil {
		t.Fatalf("generate ca cert: %v", err)
	}

	caBlock, _ := pem.Decode(caCertPEM)
	if caBlock == nil {
		t.Fatal("decode ca cert pem failed")
	}
	caCertX509, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatalf("parse ca cert: %v", err)
	}

	keyBlock, _ := pem.Decode(caKeyPEM)
	if keyBlock == nil {
		t.Fatal("decode ca key pem failed")
	}
	caPrivKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		t.Fatalf("parse ca key: %v", err)
	}

	serverCertPEM, serverKeyPEM, err := certs.GenerateChildCert("127.0.0.1", false, caCertX509, caPrivKey)
	if err != nil {
		t.Fatalf("generate server cert: %v", err)
	}

	pipeline.Tls = &clientpb.TLS{
		Enable: true,
		Cert: &clientpb.Cert{
			Cert: string(serverCertPEM),
			Key:  string(serverKeyPEM),
		},
		Ca: &clientpb.Cert{
			Cert: string(caCertPEM),
			Key:  string(caKeyPEM),
		},
	}
}

func newClientTLSConfig(tlsConfigPB *clientpb.TLS) (*tls.Config, error) {
	if tlsConfigPB == nil || !tlsConfigPB.GetEnable() {
		return nil, nil
	}

	rootCAs := x509.NewCertPool()
	if ca := tlsConfigPB.GetCa(); ca != nil && ca.GetCert() != "" {
		if ok := rootCAs.AppendCertsFromPEM([]byte(ca.GetCert())); !ok {
			return nil, errors.New("append tls ca cert failed")
		}
	}

	return &tls.Config{
		RootCAs:    rootCAs,
		MinVersion: tls.VersionTLS13,
	}, nil
}
