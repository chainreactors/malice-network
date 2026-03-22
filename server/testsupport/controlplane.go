package testsupport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/listener"
	"github.com/chainreactors/malice-network/server/rpc"
	config "github.com/gookit/config/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type ControlPlaneHarness struct {
	Address  string
	Admin    *mtls.ClientConfig
	Server   *grpc.Server
	Listener net.Listener

	control *core.Listener
	stop    chan struct{}
	wg      sync.WaitGroup

	ctrlMu      sync.Mutex
	ctrlHistory []*clientpb.JobCtrl

	failMu    sync.Mutex
	failNexts map[string]error
}

func NewControlPlaneHarness(t testing.TB) *ControlPlaneHarness {
	t.Helper()

	workdir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(workdir, ".malice"))
	if err := configs.InitConfig(); err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	serverCfg := &configs.ServerConfig{
		Enable:        true,
		GRPCHost:      "127.0.0.1",
		GRPCPort:      0,
		IP:            "127.0.0.1",
		EncryptionKey: "integration-secret",
		MiscConfig: &configs.MiscConfig{
			PacketLength: 4 * 1024 * 1024,
		},
		DatabaseConfig: &configs.DatabaseConfig{
			Dialect: configs.Sqlite,
		},
	}
	if err := configutil.SetStructByTag("server", serverCfg, "config"); err != nil {
		t.Fatalf("SetStructByTag(server) failed: %v", err)
	}
	listenerCfg := &configs.ListenerConfig{
		Enable: true,
		Name:   "fixture-listener",
		IP:     "127.0.0.1",
	}
	if err := configutil.SetStructByTag("listeners", listenerCfg, "config"); err != nil {
		t.Fatalf("SetStructByTag(listeners) failed: %v", err)
	}
	config.Set("debug", false)

	oldDBClient := db.Client
	t.Cleanup(func() {
		db.Client = oldDBClient
	})
	var dbErr error
	db.Client, dbErr = db.NewDBClient(serverCfg.DatabaseConfig)
	if dbErr != nil {
		t.Fatalf("NewDBClient failed: %v", dbErr)
	}

	if err := db.BackfillOperatorFingerprints(); err != nil {
		t.Fatalf("BackfillOperatorFingerprints failed: %v", err)
	}
	if err := db.SeedDefaultAuthzRules(); err != nil {
		t.Fatalf("SeedDefaultAuthzRules failed: %v", err)
	}

	oldTicker := core.GlobalTicker
	core.GlobalTicker = core.NewTicker()
	t.Cleanup(func() {
		core.GlobalTicker.RemoveAll()
		core.GlobalTicker = oldTicker
	})

	oldBroker := core.EventBroker
	oldSessions := core.Sessions
	oldConnections := core.Connections
	oldForwarders := core.Forwarders
	oldListenerSessions := core.ListenerSessions
	oldListenerMap := core.Listeners.Map
	oldJobsMap := core.Jobs.Map
	t.Cleanup(func() {
		core.EventBroker = oldBroker
		core.Sessions = oldSessions
		core.Connections = oldConnections
		core.Forwarders = oldForwarders
		core.ListenerSessions = oldListenerSessions
		core.Listeners.Map = oldListenerMap
		core.Jobs.Map = oldJobsMap
		for _, client := range core.Clients.ActiveClients() {
			core.Clients.Remove(int(client.ID))
		}
	})

	core.Listeners.Map = &sync.Map{}
	core.Jobs.Map = &sync.Map{}
	core.NewBroker()
	core.NewSessions()
	core.ResetTransientTransportState()
	rpc.ResetTransientRPCState()

	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}

	grpcServer, ln, err := rpc.StartClientListener("127.0.0.1:0")
	if err != nil {
		t.Fatalf("StartClientListener failed: %v", err)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	serverCfg.GRPCPort = uint16(port)
	config.Set("server.grpc_port", port)

	adminConf, fingerprint, err := certutils.GenerateClientCert(serverCfg.IP, "admin", port)
	if err != nil {
		t.Fatalf("GenerateClientCert failed: %v", err)
	}
	if err := db.CreateOperator(&models.Operator{
		Name:             "admin",
		Type:             mtls.Client,
		Role:             models.RoleAdmin,
		Fingerprint:      fingerprint,
		CAType:           certs.OperatorCA,
		KeyType:          certs.RSAKey,
		CaCertificatePEM: adminConf.CACertificate,
		CertificatePEM:   adminConf.Certificate,
		PrivateKeyPEM:    adminConf.PrivateKey,
	}); err != nil {
		t.Fatalf("CreateOperator failed: %v", err)
	}

	h := &ControlPlaneHarness{
		Address:   fmt.Sprintf("127.0.0.1:%d", port),
		Admin:     adminConf,
		Server:    grpcServer,
		Listener:  ln,
		control:   core.NewListener("fixture-listener", "127.0.0.1"),
		stop:      make(chan struct{}),
		failNexts: make(map[string]error),
	}
	core.Listeners.Add(h.control)
	h.startController()

	t.Cleanup(func() {
		close(h.stop)
		h.wg.Wait()
		core.Listeners.Remove(h.control)
		grpcServer.GracefulStop()
		_ = ln.Close()
		core.ResetTransientTransportState()
		rpc.ResetTransientRPCState()
		rpc.CloseLogs()
	})

	return h
}

func (h *ControlPlaneHarness) ListenerID() string {
	return h.control.Name
}

func (h *ControlPlaneHarness) startController() {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-h.stop:
				return
			case ctrl, ok := <-h.control.Ctrl:
				if !ok {
					return
				}
				h.handleCtrl(ctrl)
			}
		}
	}()
}

func (h *ControlPlaneHarness) recordCtrl(ctrl *clientpb.JobCtrl) {
	h.ctrlMu.Lock()
	defer h.ctrlMu.Unlock()
	if ctrl == nil {
		return
	}
	h.ctrlHistory = append(h.ctrlHistory, proto.Clone(ctrl).(*clientpb.JobCtrl))
}

func (h *ControlPlaneHarness) ControlHistory() []*clientpb.JobCtrl {
	h.ctrlMu.Lock()
	defer h.ctrlMu.Unlock()
	history := make([]*clientpb.JobCtrl, 0, len(h.ctrlHistory))
	for _, ctrl := range h.ctrlHistory {
		history = append(history, proto.Clone(ctrl).(*clientpb.JobCtrl))
	}
	return history
}

func ctrlFailureKey(ctrl, name string) string {
	return ctrl + "\x00" + name
}

func (h *ControlPlaneHarness) FailNextCtrl(ctrl, name string, err error) {
	if err == nil {
		err = errors.New("injected listener failure")
	}

	h.failMu.Lock()
	defer h.failMu.Unlock()
	h.failNexts[ctrlFailureKey(ctrl, name)] = err
}

func (h *ControlPlaneHarness) consumeCtrlFailure(ctrl *clientpb.JobCtrl) error {
	name := ctrl.GetJob().GetName()
	if name == "" && ctrl.GetJob().GetPipeline() != nil {
		name = ctrl.GetJob().GetPipeline().GetName()
	}

	h.failMu.Lock()
	defer h.failMu.Unlock()

	keys := []string{
		ctrlFailureKey(ctrl.GetCtrl(), name),
		ctrlFailureKey(ctrl.GetCtrl(), ""),
	}
	for _, key := range keys {
		if err, ok := h.failNexts[key]; ok {
			delete(h.failNexts, key)
			return err
		}
	}
	return nil
}

func (h *ControlPlaneHarness) handleCtrl(ctrl *clientpb.JobCtrl) {
	h.recordCtrl(ctrl)

	handlerErr := h.consumeCtrlFailure(ctrl)
	if handlerErr == nil {
		switch ctrl.Ctrl {
		case consts.CtrlPipelineStart:
			handlerErr = h.handlePipelineStart(ctrl)
		case consts.CtrlPipelineStop:
			handlerErr = h.handlePipelineStop(ctrl)
		case consts.CtrlRemStart:
			handlerErr = h.handleRemStart(ctrl)
		case consts.CtrlRemStop:
			handlerErr = h.handleRemStop(ctrl)
		case consts.CtrlWebsiteStart:
			handlerErr = h.handleWebsiteStart(ctrl)
		case consts.CtrlWebsiteStop:
			handlerErr = h.handleWebsiteStop(ctrl)
		case consts.CtrlWebContentAdd, consts.CtrlWebContentAddArtifact:
			handlerErr = h.handleWebsiteContentAdd(ctrl)
		case consts.CtrlWebContentRemove:
			handlerErr = h.handleWebsiteContentRemove(ctrl)
		case consts.CtrlListenerSyncSession:
		}
	}

	status := &clientpb.JobStatus{
		ListenerId: h.control.Name,
		Ctrl:       ctrl.Ctrl,
		CtrlId:     ctrl.Id,
		Job:        ctrl.Job,
	}
	if handlerErr != nil {
		status.Status = consts.CtrlStatusFailed
		status.Error = handlerErr.Error()
	} else {
		status.Status = consts.CtrlStatusSuccess
	}

	h.control.CtrlJob.Store(ctrl.Id, status)
}

func (h *ControlPlaneHarness) handlePipelineStart(ctrl *clientpb.JobCtrl) error {
	pipe := ctrl.GetJob().GetPipeline()
	if pipe == nil {
		return errors.New("pipeline is nil")
	}
	pipe.Enable = true
	core.Jobs.AddPipeline(pipe)
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlPipelineStart,
		Job:       ctrl.Job,
		Important: true,
	})
	return nil
}

func (h *ControlPlaneHarness) handlePipelineStop(ctrl *clientpb.JobCtrl) error {
	pipe := ctrl.GetJob().GetPipeline()
	if pipe == nil {
		return errors.New("pipeline is nil")
	}
	runtime := h.control.GetPipeline(pipe.Name)
	if runtime == nil {
		return errors.New("pipeline not found")
	}
	runtime.Enable = false
	h.control.RemovePipeline(runtime)
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlPipelineStop,
		Job: &clientpb.Job{
			Id:       ctrl.GetJob().GetId(),
			Name:     runtime.Name,
			Pipeline: runtime,
		},
		Important: true,
	})
	return nil
}

func (h *ControlPlaneHarness) handleWebsiteStart(ctrl *clientpb.JobCtrl) error {
	pipe := ctrl.GetJob().GetPipeline()
	if pipe == nil {
		return errors.New("pipeline is nil")
	}
	if pipe.GetWeb() != nil && pipe.GetWeb().Contents == nil {
		pipe.GetWeb().Contents = make(map[string]*clientpb.WebContent)
	}
	pipe.Enable = true
	core.Jobs.AddPipeline(pipe)
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlWebsiteStart,
		Job:       ctrl.Job,
		Important: true,
	})
	return nil
}

func (h *ControlPlaneHarness) handleWebsiteStop(ctrl *clientpb.JobCtrl) error {
	pipe := ctrl.GetJob().GetPipeline()
	if pipe == nil {
		return errors.New("pipeline is nil")
	}
	runtime := h.control.GetPipeline(pipe.Name)
	if runtime == nil {
		return errors.New("website not found")
	}
	runtime.Enable = false
	h.control.RemovePipeline(runtime)
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlWebsiteStop,
		Job: &clientpb.Job{
			Id:       ctrl.GetJob().GetId(),
			Name:     runtime.Name,
			Pipeline: runtime,
		},
		Important: true,
	})
	return nil
}

func (h *ControlPlaneHarness) handleWebsiteContentAdd(ctrl *clientpb.JobCtrl) error {
	pipe := ctrl.GetJob().GetPipeline()
	if pipe == nil {
		return errors.New("pipeline is nil")
	}
	runtime := h.control.GetPipeline(pipe.Name)
	if runtime == nil {
		return errors.New("website not found")
	}
	if runtime.GetWeb() == nil {
		return errors.New("website content container is nil")
	}
	if runtime.GetWeb().Contents == nil {
		runtime.GetWeb().Contents = make(map[string]*clientpb.WebContent)
	}
	if ctrl.Content != nil {
		runtime.GetWeb().Contents[ctrl.Content.Path] = proto.Clone(ctrl.Content).(*clientpb.WebContent)
	}
	contents := map[string]*clientpb.WebContent{}
	if ctrl.Content != nil {
		contents[ctrl.Content.Path] = proto.Clone(ctrl.Content).(*clientpb.WebContent)
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        ctrl.Ctrl,
		Job: &clientpb.Job{
			Id:       ctrl.GetJob().GetId(),
			Name:     runtime.Name,
			Pipeline: runtime,
			Contents: contents,
		},
		Important: true,
	})
	return nil
}

func (h *ControlPlaneHarness) handleWebsiteContentRemove(ctrl *clientpb.JobCtrl) error {
	pipe := ctrl.GetJob().GetPipeline()
	if pipe == nil || pipe.GetWeb() == nil {
		return errors.New("website pipeline is nil")
	}
	runtime := h.control.GetPipeline(pipe.Name)
	if runtime == nil || runtime.GetWeb() == nil {
		return errors.New("website not found")
	}
	for webPath := range pipe.GetWeb().Contents {
		delete(runtime.GetWeb().Contents, webPath)
	}
	contents := make(map[string]*clientpb.WebContent, len(pipe.GetWeb().Contents))
	for webPath, content := range pipe.GetWeb().Contents {
		if content == nil {
			content = &clientpb.WebContent{Path: webPath}
		}
		contents[webPath] = proto.Clone(content).(*clientpb.WebContent)
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlWebContentRemove,
		Job: &clientpb.Job{
			Id:       ctrl.GetJob().GetId(),
			Name:     runtime.Name,
			Pipeline: runtime,
			Contents: contents,
		},
		Important: true,
	})
	return nil
}

func (h *ControlPlaneHarness) handleRemStart(ctrl *clientpb.JobCtrl) error {
	pipe := ctrl.GetJob().GetPipeline()
	if pipe == nil {
		return errors.New("pipeline is nil")
	}
	pipe.Enable = true
	core.Jobs.AddPipeline(pipe)
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlRemStart,
		Job:       ctrl.Job,
		Important: true,
	})
	return nil
}

func (h *ControlPlaneHarness) handleRemStop(ctrl *clientpb.JobCtrl) error {
	pipe := ctrl.GetJob().GetPipeline()
	if pipe == nil {
		return errors.New("pipeline is nil")
	}
	runtime := h.control.GetPipeline(pipe.Name)
	if runtime == nil {
		return errors.New("rem not found")
	}
	runtime.Enable = false
	h.control.RemovePipeline(runtime)
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventJob,
		Op:        consts.CtrlRemStop,
		Job: &clientpb.Job{
			Id:       ctrl.GetJob().GetId(),
			Name:     runtime.Name,
			Pipeline: runtime,
		},
		Important: true,
	})
	return nil
}

func (h *ControlPlaneHarness) NewTCPPipeline(t testing.TB, name string) *clientpb.Pipeline {
	t.Helper()

	pipeline, err := (&configs.TcpPipelineConfig{
		Enable: true,
		Name:   name,
		Host:   "127.0.0.1",
		Port:   0,
		Parser: consts.ImplantMalefic,
		EncryptionConfig: implanttypes.EncryptionsConfig{
			{Type: "xor", Key: "integration-secret"},
		},
	}).ToProtobuf(h.control.Name)
	if err != nil {
		t.Fatalf("TcpPipelineConfig.ToProtobuf failed: %v", err)
	}
	return pipeline
}

func (h *ControlPlaneHarness) NewHTTPPipeline(t testing.TB, name string) *clientpb.Pipeline {
	t.Helper()

	pipeline, err := (&configs.HttpPipelineConfig{
		Enable: true,
		Name:   name,
		Host:   "127.0.0.1",
		Port:   0,
		Parser: consts.ImplantMalefic,
		EncryptionConfig: implanttypes.EncryptionsConfig{
			{Type: "xor", Key: "integration-secret"},
		},
		Headers: map[string][]string{
			"X-Test": {"integration"},
		},
	}).ToProtobuf(h.control.Name)
	if err != nil {
		t.Fatalf("HttpPipelineConfig.ToProtobuf failed: %v", err)
	}
	return pipeline
}

func (h *ControlPlaneHarness) NewBindPipeline(t testing.TB, name string) *clientpb.Pipeline {
	t.Helper()

	pipeline, err := (&configs.BindPipelineConfig{
		Enable: true,
		Name:   name,
	}).ToProtobuf(h.control.Name)
	if err != nil {
		t.Fatalf("BindPipelineConfig.ToProtobuf failed: %v", err)
	}
	return pipeline
}

func (h *ControlPlaneHarness) NewREMPipeline(name, console string) *clientpb.Pipeline {
	if console == "" {
		console = "tcp://127.0.0.1:19966"
	}
	return &clientpb.Pipeline{
		Name:       name,
		ListenerId: h.control.Name,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:      name,
				Host:      "127.0.0.1",
				Port:      19966,
				Console:   console,
				Agents:    make(map[string]*clientpb.REMAgent),
				Link:      "tcp://127.0.0.1:19966",
				Subscribe: "pivot",
			},
		},
	}
}

func (h *ControlPlaneHarness) NewWebsitePipeline(name string, port uint32, root string) *clientpb.Pipeline {
	if root == "" {
		root = "/"
	}
	return &clientpb.Pipeline{
		Name:       name,
		ListenerId: h.control.Name,
		Type:       consts.WebsitePipeline,
		Ip:         h.control.IP,
		Tls:        &clientpb.TLS{},
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:       name,
				ListenerId: h.control.Name,
				Root:       root,
				Port:       port,
				Contents:   make(map[string]*clientpb.WebContent),
			},
		},
	}
}

func (h *ControlPlaneHarness) SeedPipeline(t testing.TB, pipeline *clientpb.Pipeline, started bool) *clientpb.Pipeline {
	t.Helper()

	if pipeline == nil {
		t.Fatal("pipeline is nil")
	}
	pipeline.Enable = started
	pipeline.Ip = h.control.IP
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	if started {
		core.Jobs.AddPipeline(proto.Clone(pipeline).(*clientpb.Pipeline))
	}
	return pipeline
}

func (h *ControlPlaneHarness) GetPipeline(name, listenerID string) (*clientpb.Pipeline, error) {
	model, err := db.FindPipelineByListener(name, listenerID)
	if err != nil {
		return nil, err
	}
	return model.ToProtobuf(), nil
}

func (h *ControlPlaneHarness) SeedWebsite(t testing.TB, pipeline *clientpb.Pipeline, started bool) *clientpb.Pipeline {
	t.Helper()

	if pipeline == nil || pipeline.GetWeb() == nil {
		t.Fatal("website pipeline is nil")
	}
	if pipeline.Tls == nil {
		pipeline.Tls = &clientpb.TLS{}
	}
	pipeline.Enable = started
	pipeline.Ip = h.control.IP
	if _, err := db.SavePipeline(models.FromPipelinePb(pipeline)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}
	for _, content := range pipeline.GetWeb().Contents {
		content.WebsiteId = pipeline.Name
		if _, err := db.AddContent(content); err != nil {
			t.Fatalf("AddContent failed: %v", err)
		}
	}
	if started {
		core.Jobs.AddPipeline(proto.Clone(pipeline).(*clientpb.Pipeline))
	}
	return pipeline
}

func (h *ControlPlaneHarness) GetWebsite(name string) (*clientpb.Pipeline, error) {
	model, err := db.FindWebsiteByName(name)
	if err != nil {
		return nil, err
	}
	return model.ToProtobuf(), nil
}

func (h *ControlPlaneHarness) GetWebContents(name string) ([]*clientpb.WebContent, error) {
	contents, err := db.FindWebContentsByWebsite(name)
	if err != nil {
		return nil, err
	}
	ret := make([]*clientpb.WebContent, 0, len(contents))
	for _, content := range contents {
		ret = append(ret, content.ToProtobuf(false))
	}
	return ret, nil
}

func (h *ControlPlaneHarness) GetWebContent(id string) (*clientpb.WebContent, error) {
	content, err := db.FindWebContent(id)
	if err != nil {
		return nil, err
	}
	return content.ToProtobuf(false), nil
}

func (h *ControlPlaneHarness) ReadWebsiteContent(websiteName, contentID string) ([]byte, error) {
	contentPath := filepath.Join(configs.WebsitePath, websiteName, contentID)
	return os.ReadFile(contentPath)
}

// StartRealWebsite reads a website's pipeline and content from DB, starts a real
// HTTP server on a random port, and returns the base URL. The server is stopped
// on test cleanup. This bridges the gap between the mock control plane and actual
// HTTP serving for full E2E verification.
func (h *ControlPlaneHarness) StartRealWebsite(t testing.TB, name string) string {
	t.Helper()

	pipe, err := db.FindWebsiteByName(name)
	if err != nil {
		t.Fatalf("FindWebsiteByName(%s): %v", name, err)
	}
	pipeProto := pipe.ToProtobuf()

	// Override port to 0 so OS picks a free port.
	if pipeProto.GetWeb() != nil {
		pipeProto.GetWeb().Port = 0
	}

	dbContents, err := db.FindWebContentsByWebsite(name)
	if err != nil {
		t.Fatalf("FindWebContentsByWebsite(%s): %v", name, err)
	}
	contentMap := make(map[string]*clientpb.WebContent)
	for _, c := range dbContents {
		pb := c.ToProtobuf(true) // include content bytes
		contentMap[pb.Path] = pb
	}

	web, err := listener.StartWebsite(nil, pipeProto, contentMap)
	if err != nil {
		t.Fatalf("StartWebsite(%s): %v", name, err)
	}
	t.Cleanup(func() { web.Close() })

	addr := web.Addr()
	return fmt.Sprintf("http://127.0.0.1:%d", addr.Port)
}

func (h *ControlPlaneHarness) JobExists(name, listenerID string) bool {
	_, err := core.Jobs.GetByListener(name, listenerID)
	return err == nil
}

func (h *ControlPlaneHarness) SeedSession(t testing.TB, sessionID, pipelineName string, active bool) *core.Session {
	t.Helper()

	if pipelineName == "" {
		pipelineName = "seed-pipeline"
		h.SeedPipeline(t, h.NewTCPPipeline(t, pipelineName), true)
	}

	req := &clientpb.RegisterSession{
		Type:       consts.TCPPipeline,
		SessionId:  sessionID,
		RawId:      1,
		PipelineId: pipelineName,
		ListenerId: h.control.Name,
		Target:     "127.0.0.1",
		RegisterData: &implantpb.Register{
			Name: "seed-artifact",
			Timer: &implantpb.Timer{
				Expression: "* * * * *",
			},
			Sysinfo: &implantpb.SysInfo{
				Os: &implantpb.Os{
					Name: "windows",
					Arch: "amd64",
				},
				Process: &implantpb.Process{
					Name: "seed.exe",
				},
			},
		},
	}

	sess, err := core.RegisterSession(req)
	if err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}
	sess.SetLastCheckin(time.Now().Unix())
	if err := sess.Save(); err != nil {
		t.Fatalf("session.Save failed: %v", err)
	}
	if active {
		core.Sessions.Add(sess)
	}
	return sess
}

func (h *ControlPlaneHarness) GetSession(sessionID string) (*clientpb.Session, error) {
	model, err := db.FindSession(sessionID)
	if err != nil || model == nil {
		return nil, err
	}
	return model.ToProtobuf(), nil
}

func (h *ControlPlaneHarness) SeedTask(t testing.TB, sess *core.Session, typ string) *core.Task {
	t.Helper()

	if sess == nil {
		t.Fatal("session is nil")
	}
	if typ == "" {
		typ = "seed-task"
	}
	task := sess.NewTask(typ, 1)
	task.Cur = 1
	task.CreatedAt = time.Now()
	task.FinishedAt = task.CreatedAt
	task.CallBy = consts.CalleeCMD
	if err := db.AddTask(task.ToProtobuf()); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := db.UpdateTaskFinish(task.TaskID()); err != nil {
		t.Fatalf("UpdateTaskFinish failed: %v", err)
	}
	return task
}

func (h *ControlPlaneHarness) SeedDownloadContext(t testing.TB, task *core.Task, fileName string, content []byte) *clientpb.Context {
	t.Helper()

	if task == nil || task.Session == nil {
		t.Fatal("task or session is nil")
	}
	filePath, err := h.WriteTempFile(fileName, content)
	if err != nil {
		t.Fatalf("WriteTempFile failed: %v", err)
	}
	ctx, err := core.SaveContext(&output.DownloadContext{
		FileDescriptor: &output.FileDescriptor{
			Name:       filepath.Base(filePath),
			TargetPath: "remote/path",
			FilePath:   filePath,
			Size:       int64(len(content)),
		},
	}, task)
	if err != nil {
		t.Fatalf("SaveContext failed: %v", err)
	}
	return ctx.ToProtobuf()
}

func (h *ControlPlaneHarness) SeedCredentialContext(t testing.TB, task *core.Task, target string, params map[string]string) *clientpb.Context {
	t.Helper()

	if task == nil || task.Session == nil {
		t.Fatal("task or session is nil")
	}
	ctx, err := core.SaveContext(&output.CredentialContext{
		CredentialType: output.UserPassCredential,
		Target:         target,
		Params:         params,
	}, task)
	if err != nil {
		t.Fatalf("SaveContext failed: %v", err)
	}
	return ctx.ToProtobuf()
}

func (h *ControlPlaneHarness) GetContext(id string) (*clientpb.Context, error) {
	model, err := db.FindContext(id)
	if err != nil {
		return nil, err
	}
	return model.ToProtobuf(), nil
}

func (h *ControlPlaneHarness) WriteTempFile(name string, content []byte) (string, error) {
	if name == "" {
		name = "fixture.bin"
	}
	filePath := filepath.Join(configs.TempPath, name)
	if err := os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		return "", err
	}
	return filePath, nil
}

func (h *ControlPlaneHarness) Connect(ctx context.Context) (*grpc.ClientConn, error) {
	return h.ConnectWithConfig(ctx, h.Admin)
}

func (h *ControlPlaneHarness) ConnectWithConfig(ctx context.Context, config *mtls.ClientConfig) (*grpc.ClientConn, error) {
	if config == nil {
		return nil, errors.New("client config is nil")
	}
	options, err := mtls.GetGrpcOptions(
		[]byte(config.CACertificate),
		[]byte(config.Certificate),
		[]byte(config.PrivateKey),
		config.Type,
	)
	if err != nil {
		return nil, err
	}
	return grpc.DialContext(ctx, h.Address, options...)
}

func (h *ControlPlaneHarness) NewListenerClientConfig(t testing.TB, name string) *mtls.ClientConfig {
	t.Helper()

	if name == "" {
		t.Fatal("listener name is empty")
	}

	cfg := configs.GetServerConfig()
	clientConf, fingerprint, err := certutils.GenerateListenerCert(cfg.IP, name, int(cfg.GRPCPort))
	if err != nil {
		t.Fatalf("GenerateListenerCert failed: %v", err)
	}

	if err := db.CreateOperator(&models.Operator{
		Name:             name,
		Type:             mtls.Listener,
		Role:             models.RoleListener,
		Fingerprint:      fingerprint,
		CAType:           certs.ListenerCA,
		KeyType:          certs.RSAKey,
		CaCertificatePEM: clientConf.CACertificate,
		CertificatePEM:   clientConf.Certificate,
		PrivateKeyPEM:    clientConf.PrivateKey,
	}); err != nil {
		t.Fatalf("CreateOperator(listener) failed: %v", err)
	}

	return clientConf
}

func WaitForCondition(t testing.TB, timeout time.Duration, cond func() bool, description string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}

func WaitForEvent(t testing.TB, ch <-chan *clientpb.Event, description string) *clientpb.Event {
	t.Helper()

	select {
	case event := <-ch:
		return event
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
		return nil
	}
}
