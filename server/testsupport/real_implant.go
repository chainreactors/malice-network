//go:build realimplant

package testsupport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/listener"
	serverrpc "github.com/chainreactors/malice-network/server/rpc"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

const (
	defaultRealImplantWorkspace = `D:\Programing\rust\implant`
	defaultRealImplantTemplate  = `D:\Programing\rust\implant\target\debug\malefic.exe`
	defaultRealImplantMutant    = `D:\Programing\rust\implant\target\debug\malefic-mutant.exe`
)

type RealImplantEnv struct {
	Workspace    string
	TemplatePath string
	MutantPath   string
}

type RealImplant struct {
	Harness  *ControlPlaneHarness
	Pipeline *clientpb.Pipeline

	ListenerName string
	SessionID    string
	SessionName  string

	ProfilePath string
	BinaryPath  string

	cmd     *exec.Cmd
	stdout  bytes.Buffer
	stderr  bytes.Buffer
	waitErr error
	waitCh  chan struct{}
	waitMu  sync.Mutex
}

func ResolveRealImplantEnv() (RealImplantEnv, error) {
	env := RealImplantEnv{
		Workspace:    strings.TrimSpace(os.Getenv("MALICE_REAL_IMPLANT_WORKSPACE")),
		TemplatePath: strings.TrimSpace(os.Getenv("MALICE_REAL_IMPLANT_BIN")),
		MutantPath:   strings.TrimSpace(os.Getenv("MALICE_REAL_IMPLANT_MUTANT")),
	}
	if env.Workspace == "" {
		env.Workspace = defaultRealImplantWorkspace
	}
	if env.TemplatePath == "" {
		env.TemplatePath = defaultRealImplantTemplate
	}
	if env.MutantPath == "" {
		env.MutantPath = defaultRealImplantMutant
	}

	if _, err := os.Stat(env.TemplatePath); err != nil {
		return env, fmt.Errorf("real implant template not found: %s", env.TemplatePath)
	}
	if _, err := os.Stat(env.MutantPath); err != nil {
		return env, fmt.Errorf("real implant mutant not found: %s", env.MutantPath)
	}
	return env, nil
}

func RequireRealImplantEnv(t testing.TB) RealImplantEnv {
	t.Helper()

	if strings.TrimSpace(os.Getenv("MALICE_REAL_IMPLANT_RUN")) == "" {
		t.Skip("set MALICE_REAL_IMPLANT_RUN=1 to enable real implant integration tests")
	}

	env, err := ResolveRealImplantEnv()
	if err != nil {
		t.Skip(err.Error())
	}
	return env
}

func NewRealTCPPipeline(t testing.TB, listenerName, pipelineName string) *clientpb.Pipeline {
	t.Helper()

	port := reserveTCPPort(t)
	return &clientpb.Pipeline{
		Name:       pipelineName,
		ListenerId: listenerName,
		Enable:     true,
		Parser:     consts.ImplantMalefic,
		Type:       consts.TCPPipeline,
		Ip:         "127.0.0.1",
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       pipelineName,
				ListenerId: listenerName,
				Host:       "127.0.0.1",
				Port:       uint32(port),
			},
		},
		Tls: &clientpb.TLS{},
		Encryption: []*clientpb.Encryption{
			{
				Type: consts.CryptorAES,
				Key:  "integration-secret",
			},
		},
	}
}

func NewRealImplant(t testing.TB, h *ControlPlaneHarness, pipeline *clientpb.Pipeline) *RealImplant {
	t.Helper()

	if h == nil {
		t.Fatal("control plane harness is nil")
	}
	if pipeline == nil {
		t.Fatal("pipeline is nil")
	}

	clone := proto.Clone(pipeline).(*clientpb.Pipeline)
	if clone.ListenerId == "" {
		t.Fatal("pipeline listener id is empty")
	}
	if clone.Name == "" {
		t.Fatal("pipeline name is empty")
	}

	ri := &RealImplant{
		Harness:      h,
		Pipeline:     clone,
		ListenerName: clone.ListenerId,
		SessionName:  fmt.Sprintf("real-implant-%d", time.Now().UnixNano()),
		waitCh:       make(chan struct{}),
	}

	t.Cleanup(func() {
		_ = ri.Close()
	})
	return ri
}

func (r *RealImplant) Start(t testing.TB) error {
	t.Helper()

	if r == nil {
		return errors.New("real implant is nil")
	}
	if r.cmd != nil {
		return nil
	}

	env := RequireRealImplantEnv(t)
	if err := r.startListenerAndPipeline(t); err != nil {
		return err
	}
	if err := r.generatePatchedBinary(t, env); err != nil {
		return err
	}
	if err := r.startProcess(); err != nil {
		return err
	}

	sessionID, err := r.waitForSession(30 * time.Second)
	if err != nil {
		return err
	}
	r.SessionID = sessionID
	return nil
}

func (r *RealImplant) Close() error {
	if r == nil {
		return nil
	}

	var errs []string

	if r.cmd != nil && r.cmd.Process != nil {
		select {
		case <-r.waitCh:
		default:
			if err := r.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				errs = append(errs, fmt.Sprintf("kill implant process: %v", err))
			}
		}
		select {
		case <-r.waitCh:
		case <-time.After(5 * time.Second):
			errs = append(errs, "timed out waiting for implant process exit")
		}
		r.cmd = nil
	}

	if r.Pipeline != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := (&serverrpc.Server{}).StopPipeline(ctx, &clientpb.CtrlPipeline{
			Name:       r.Pipeline.Name,
			ListenerId: r.ListenerName,
			Pipeline:   proto.Clone(r.Pipeline).(*clientpb.Pipeline),
		})
		cancel()
		if err != nil {
			errs = append(errs, fmt.Sprintf("stop pipeline: %v", err))
		}
	}

	if listener.Listener != nil {
		if err := listener.Listener.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("close listener: %v", err))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (r *RealImplant) startListenerAndPipeline(t testing.TB) error {
	authConfig := r.Harness.NewListenerClientConfig(t, r.ListenerName)
	authBytes, err := yaml.Marshal(authConfig)
	if err != nil {
		return fmt.Errorf("marshal listener auth: %w", err)
	}

	authPath, err := r.Harness.WriteTempFile(r.ListenerName+".auth", authBytes)
	if err != nil {
		return fmt.Errorf("write listener auth: %w", err)
	}

	cfg := &configs.ListenerConfig{
		Enable: true,
		Name:   r.ListenerName,
		Auth:   authPath,
		IP:     "127.0.0.1",
	}
	if err := listener.NewListener(authConfig, cfg, true); err != nil {
		return fmt.Errorf("start in-process listener %s: %w", r.ListenerName, err)
	}

	r.Pipeline.ListenerId = r.ListenerName
	r.Pipeline.Ip = "127.0.0.1"

	if _, err := (&serverrpc.Server{}).RegisterPipeline(context.Background(), proto.Clone(r.Pipeline).(*clientpb.Pipeline)); err != nil {
		return fmt.Errorf("register pipeline %s: %w", r.Pipeline.Name, err)
	}
	if _, err := (&serverrpc.Server{}).StartPipeline(context.Background(), &clientpb.CtrlPipeline{
		Name:       r.Pipeline.Name,
		ListenerId: r.ListenerName,
		Pipeline:   proto.Clone(r.Pipeline).(*clientpb.Pipeline),
	}); err != nil {
		return fmt.Errorf("start pipeline %s: %w", r.Pipeline.Name, err)
	}
	return nil
}

func (r *RealImplant) generatePatchedBinary(t testing.TB, env RealImplantEnv) error {
	t.Helper()

	profile, err := models.FromPipelinePb(r.Pipeline).ToProfile(nil)
	if err != nil {
		return fmt.Errorf("build implant profile from pipeline: %w", err)
	}
	profile.Basic.Name = r.SessionName
	profile.Basic.Cron = "*/1 * * * * * *"
	profile.Basic.Jitter = 0
	profile.Basic.Keepalive = true
	profile.Basic.Retry = 5
	profile.Basic.MaxCycles = -1
	profile.Implant.Mod = "beacon"
	profile.Implant.Runtime = "tokio"
	profile.Implant.RegisterInfo = true
	profile.Implant.HotLoad = true

	profileYAML, err := profile.ToYAML()
	if err != nil {
		return fmt.Errorf("marshal implant profile: %w", err)
	}
	profilePath, err := r.Harness.WriteTempFile(r.SessionName+"-implant.yaml", profileYAML)
	if err != nil {
		return fmt.Errorf("write implant profile: %w", err)
	}
	r.ProfilePath = profilePath

	outputPath := filepath.Join(configs.TempPath, r.SessionName+".exe")
	cmd := exec.Command(
		env.MutantPath,
		"tool", "patch-config",
		"-f", env.TemplatePath,
		"--from-implant", profilePath,
		"-o", outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("patch real implant config failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	r.BinaryPath = outputPath
	return nil
}

func (r *RealImplant) startProcess() error {
	cmd := exec.Command(r.BinaryPath)
	cmd.Dir = filepath.Dir(r.BinaryPath)
	cmd.Stdout = &r.stdout
	cmd.Stderr = &r.stderr
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start real implant process: %w", err)
	}

	r.cmd = cmd
	go func() {
		r.waitMu.Lock()
		r.waitErr = cmd.Wait()
		r.waitMu.Unlock()
		close(r.waitCh)
	}()
	return nil
}

func (r *RealImplant) waitForSession(timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if session := r.findRuntimeSession(); session != nil {
			return session.ID, nil
		}
		select {
		case <-r.waitCh:
			return "", fmt.Errorf(
				"real implant exited before registering: %v\nstdout:\n%s\nstderr:\n%s",
				r.processWaitErr(),
				strings.TrimSpace(r.stdout.String()),
				strings.TrimSpace(r.stderr.String()),
			)
		default:
		}
		time.Sleep(100 * time.Millisecond)
	}

	return "", fmt.Errorf(
		"timed out waiting for real implant session on pipeline %s\nstdout:\n%s\nstderr:\n%s",
		r.Pipeline.Name,
		strings.TrimSpace(r.stdout.String()),
		strings.TrimSpace(r.stderr.String()),
	)
}

func (r *RealImplant) findRuntimeSession() *core.Session {
	for _, session := range core.Sessions.All() {
		if session == nil {
			continue
		}
		if session.PipelineID != r.Pipeline.Name {
			continue
		}
		if r.SessionName != "" && session.Name != r.SessionName {
			continue
		}
		return session
	}
	return nil
}

func (r *RealImplant) processWaitErr() error {
	r.waitMu.Lock()
	defer r.waitMu.Unlock()
	return r.waitErr
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
