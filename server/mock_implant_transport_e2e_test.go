//go:build mockimplant

package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	iomtypes "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/testsupport"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func runtimeConnectionInactive(sessionID string) bool {
	conn := core.Connections.Get(sessionID)
	return conn == nil || !conn.IsAlive()
}

func TestMockImplantParserListenerTCPE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(testsupport.ReserveTCPPort(t))

	testsupport.StartInProcessListener(t, h, listenerName)
	testsupport.StartPipeline(t, pipeline)

	implant := testsupport.NewTCPPacketImplant(t, pipeline)
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
	if got := session.GetOs().GetHostname(); got != "tcp-mock-host" {
		t.Fatalf("session hostname = %q, want tcp-mock-host", got)
	}

	testsupport.DrainTCPServerPackets(t, implant, 200*time.Millisecond)

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

	request := testsupport.WaitForTCPModuleRequest(t, implant, consts.ModulePwd, 5*time.Second)
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

	content := testsupport.WaitTaskFinish(t, rpc, implant.SessionID, task.TaskId)
	if got := content.GetSpite().GetResponse().GetOutput(); got != `C:\integration\work` {
		t.Fatalf("pwd output = %q, want %q", got, `C:\integration\work`)
	}
}

func TestMockImplantParserListenerTCPAESE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-aes-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-aes-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(testsupport.ReserveTCPPort(t))
	pipeline.Encryption = []*clientpb.Encryption{
		{
			Type: consts.CryptorAES,
			Key:  "integration-secret-aes",
		},
	}

	testsupport.StartInProcessListener(t, h, listenerName)
	testsupport.StartPipeline(t, pipeline)

	implant := testsupport.NewTCPPacketImplant(t, pipeline)
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

	request := testsupport.WaitForTCPModuleRequest(t, implant, consts.ModulePwd, 5*time.Second)
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

	content := testsupport.WaitTaskFinish(t, rpc, implant.SessionID, task.TaskId)
	if got := content.GetSpite().GetResponse().GetOutput(); got != `C:\integration\aes` {
		t.Fatalf("aes pwd output = %q, want %q", got, `C:\integration\aes`)
	}
}

func TestMockImplantParserListenerTCPTLSE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-tls-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-tls-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(testsupport.ReserveTCPPort(t))
	testsupport.EnableSelfSignedTLS(t, pipeline)

	testsupport.StartInProcessListener(t, h, listenerName)
	testsupport.StartPipeline(t, pipeline)

	implant := testsupport.NewTCPPacketImplant(t, pipeline)
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

	request := testsupport.WaitForTCPModuleRequest(t, implant, consts.ModulePwd, 5*time.Second)
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

	content := testsupport.WaitTaskFinish(t, rpc, implant.SessionID, task.TaskId)
	if got := content.GetSpite().GetResponse().GetOutput(); got != `C:\integration\tls` {
		t.Fatalf("tls pwd output = %q, want %q", got, `C:\integration\tls`)
	}
}

func TestMockImplantParserListenerRejectsMalformedTCPPacket(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-bad-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-bad-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(testsupport.ReserveTCPPort(t))

	testsupport.StartInProcessListener(t, h, listenerName)
	testsupport.StartPipeline(t, pipeline)

	implant := testsupport.NewTCPPacketImplant(t, pipeline)
	malformed := make([]byte, 9)
	malformed[0] = 0x00
	binary.LittleEndian.PutUint32(malformed[1:5], implant.RawID)
	binary.LittleEndian.PutUint32(malformed[5:9], 0)

	if err := implant.WritePlain(malformed); err != nil {
		t.Fatalf("send malformed packet failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return runtimeConnectionInactive(implant.SessionID)
	}, "runtime connection cleanup for malformed packet")

	if _, err := core.Sessions.Get(implant.SessionID); err == nil {
		t.Fatalf("session %s should not be registered from malformed packet", implant.SessionID)
	}
	if session, err := h.GetSession(implant.SessionID); err == nil && session != nil {
		t.Fatalf("database session = %#v, want nil for malformed packet", session)
	}
	if conn := core.Connections.Get(implant.SessionID); conn != nil && conn.IsAlive() {
		t.Fatalf("runtime connection should be inactive after malformed packet, got %#v", conn)
	}
}

func TestMockImplantParserListenerRejectsWrongEncryptionKey(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-crypt-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-crypt-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(testsupport.ReserveTCPPort(t))

	testsupport.StartInProcessListener(t, h, listenerName)
	testsupport.StartPipeline(t, pipeline)

	wrongPipeline := proto.Clone(pipeline).(*clientpb.Pipeline)
	wrongPipeline.Encryption = []*clientpb.Encryption{
		{
			Type: pipeline.GetEncryption()[0].GetType(),
			Key:  "wrong-integration-secret",
		},
	}
	implant := testsupport.NewTCPPacketImplant(t, wrongPipeline)
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
		return runtimeConnectionInactive(implant.SessionID)
	}, "runtime connection cleanup for wrong encryption")

	if _, err := core.Sessions.Get(implant.SessionID); err == nil {
		t.Fatalf("session %s should not be registered with wrong encryption", implant.SessionID)
	}
	if session, err := h.GetSession(implant.SessionID); err == nil && session != nil {
		t.Fatalf("database session = %#v, want nil for wrong encryption", session)
	}
	if conn := core.Connections.Get(implant.SessionID); conn != nil && conn.IsAlive() {
		t.Fatalf("runtime connection should be inactive after wrong encryption, got %#v", conn)
	}
}

func TestMockImplantParserListenerRejectsInvalidPayloadTerminator(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-listener-payload-%d", time.Now().UnixNano())
	pipeline := h.NewTCPPipeline(t, "mock-parser-payload-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetTcp().Port = uint32(testsupport.ReserveTCPPort(t))

	testsupport.StartInProcessListener(t, h, listenerName)
	testsupport.StartPipeline(t, pipeline)

	implant := testsupport.NewTCPPacketImplant(t, pipeline)
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
		return runtimeConnectionInactive(implant.SessionID)
	}, "runtime connection cleanup for invalid payload terminator")

	if _, err := core.Sessions.Get(implant.SessionID); err == nil {
		t.Fatalf("session %s should not be registered from invalid payload", implant.SessionID)
	}
	if session, err := h.GetSession(implant.SessionID); err == nil && session != nil {
		t.Fatalf("database session = %#v, want nil for invalid payload", session)
	}
	if conn := core.Connections.Get(implant.SessionID); conn != nil && conn.IsAlive() {
		t.Fatalf("runtime connection should be inactive after invalid payload, got %#v", conn)
	}
}

func TestMockImplantParserListenerHTTPE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-http-listener-%d", time.Now().UnixNano())
	pipeline := h.NewHTTPPipeline(t, "mock-parser-http-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetHttp().Port = uint32(testsupport.ReserveTCPPort(t))
	pipeline.GetHttp().Params = (&implanttypes.PipelineParams{
		Headers:    map[string][]string{"X-Test": {"integration"}},
		BodyPrefix: "prefix:",
		BodySuffix: ":suffix",
	}).String()

	testsupport.StartInProcessListener(t, h, listenerName)
	testsupport.StartPipeline(t, pipeline)

	implant := testsupport.NewHTTPPacketImplant(t, pipeline)
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

	request, requestResp, err := testsupport.WaitForHTTPModuleRequest(implant, consts.ModulePwd, 5*time.Second)
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

	content := testsupport.WaitTaskFinish(t, rpc, implant.SessionID, task.TaskId)
	if got := content.GetSpite().GetResponse().GetOutput(); got != `C:\http\work` {
		t.Fatalf("pwd output = %q, want %q", got, `C:\http\work`)
	}
}

func TestMockImplantParserListenerHTTPRejectsMalformedPacket(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-http-bad-listener-%d", time.Now().UnixNano())
	pipeline := h.NewHTTPPipeline(t, "mock-parser-http-bad-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetHttp().Port = uint32(testsupport.ReserveTCPPort(t))
	pipeline.GetHttp().Params = (&implanttypes.PipelineParams{
		ErrorPage: "<html>bad-request</html>",
	}).String()

	testsupport.StartInProcessListener(t, h, listenerName)
	testsupport.StartPipeline(t, pipeline)

	implant := testsupport.NewHTTPPacketImplant(t, pipeline)
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

	if _, err := core.Sessions.Get(implant.SessionID); err == nil {
		t.Fatalf("session %s should not be registered from malformed http packet", implant.SessionID)
	}
	if session, err := h.GetSession(implant.SessionID); err == nil && session != nil {
		t.Fatalf("database session = %#v, want nil for malformed http packet", session)
	}
}

func TestMockImplantParserListenerHTTPRejectsWrongEncryptionKey(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-http-crypt-listener-%d", time.Now().UnixNano())
	pipeline := h.NewHTTPPipeline(t, "mock-parser-http-crypt-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetHttp().Port = uint32(testsupport.ReserveTCPPort(t))

	testsupport.StartInProcessListener(t, h, listenerName)
	testsupport.StartPipeline(t, pipeline)

	wrongPipeline := proto.Clone(pipeline).(*clientpb.Pipeline)
	wrongPipeline.Encryption = []*clientpb.Encryption{
		{
			Type: pipeline.GetEncryption()[0].GetType(),
			Key:  "wrong-http-secret",
		},
	}
	implant := testsupport.NewHTTPPacketImplant(t, wrongPipeline)
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

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return runtimeConnectionInactive(implant.SessionID)
	}, "http runtime connection cleanup for wrong encryption")

	if _, err := core.Sessions.Get(implant.SessionID); err == nil {
		t.Fatalf("session %s should not be registered with wrong http encryption", implant.SessionID)
	}
	if session, err := h.GetSession(implant.SessionID); err == nil && session != nil {
		t.Fatalf("database session = %#v, want nil for wrong http encryption", session)
	}
	if conn := core.Connections.Get(implant.SessionID); conn != nil && conn.IsAlive() {
		t.Fatalf("runtime connection should be inactive for wrong http encryption, got %#v", conn)
	}
}

func TestMockImplantParserListenerHTTPTLSE2E(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	listenerName := fmt.Sprintf("mock-parser-http-tls-listener-%d", time.Now().UnixNano())
	pipeline := h.NewHTTPPipeline(t, "mock-parser-http-tls-pipe")
	pipeline.ListenerId = listenerName
	pipeline.GetHttp().Port = uint32(testsupport.ReserveTCPPort(t))
	pipeline.GetHttp().Params = (&implanttypes.PipelineParams{
		Headers: map[string][]string{"X-TLS": {"true"}},
	}).String()
	testsupport.EnableSelfSignedTLS(t, pipeline)

	testsupport.StartInProcessListener(t, h, listenerName)
	testsupport.StartPipeline(t, pipeline)

	implant := testsupport.NewHTTPPacketImplant(t, pipeline)
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

	request, requestResp, err := testsupport.WaitForHTTPModuleRequest(implant, consts.ModulePwd, 5*time.Second)
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

	content := testsupport.WaitTaskFinish(t, rpc, implant.SessionID, task.TaskId)
	if got := content.GetSpite().GetResponse().GetOutput(); got != `C:\http\tls` {
		t.Fatalf("https pwd output = %q, want %q", got, `C:\http\tls`)
	}
}
