package listener

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc"
)

type queuedForwardStream struct {
	reqs chan *clientpb.SpiteRequest
	once sync.Once
}

func newQueuedForwardStream() *queuedForwardStream {
	return &queuedForwardStream{
		reqs: make(chan *clientpb.SpiteRequest, 4),
	}
}

func (s *queuedForwardStream) Send(*clientpb.SpiteResponse) error {
	return nil
}

func (s *queuedForwardStream) Recv() (*clientpb.SpiteRequest, error) {
	req, ok := <-s.reqs
	if !ok {
		return nil, io.EOF
	}
	return req, nil
}

func (s *queuedForwardStream) enqueue(req *clientpb.SpiteRequest) {
	s.reqs <- req
}

func (s *queuedForwardStream) close() {
	s.once.Do(func() {
		close(s.reqs)
	})
}

type queuedPipelineRPCClient struct {
	stream *queuedForwardStream
}

func (c *queuedPipelineRPCClient) OpenForwardStream(context.Context, core.Pipeline) (core.ForwardStream, error) {
	return c.stream, nil
}

func (c *queuedPipelineRPCClient) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

func (c *queuedPipelineRPCClient) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

func (c *queuedPipelineRPCClient) GetArtifact(context.Context, *clientpb.Artifact, ...grpc.CallOption) (*clientpb.Artifact, error) {
	return &clientpb.Artifact{}, nil
}

func setupForwardDispatchTest(t testing.TB) {
	t.Helper()

	configs.InitTestConfigRuntime(t)
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), ".malice"))
	if err := os.MkdirAll(configs.ServerRootPath, 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	oldDBClient := db.Client
	t.Cleanup(func() {
		db.Client = oldDBClient
	})
	var dbErr error
	db.Client, dbErr = db.NewDBClient(nil)
	if dbErr != nil {
		t.Fatalf("NewDBClient failed: %v", dbErr)
	}

	oldTicker := core.GlobalTicker
	core.GlobalTicker = core.NewTicker()
	t.Cleanup(func() {
		core.GlobalTicker.RemoveAll()
		core.GlobalTicker = oldTicker
	})

	oldBroker := core.EventBroker
	oldSessions := core.Sessions
	oldListenerMap := core.Listeners.Map
	oldJobsMap := core.Jobs.Map
	t.Cleanup(func() {
		if core.EventBroker != nil {
			core.EventBroker.Stop()
		}
		core.EventBroker = oldBroker
		core.Sessions = oldSessions
		core.Listeners.Map = oldListenerMap
		core.Jobs.Map = oldJobsMap
		core.ResetTransientTransportState()
	})

	core.Listeners.Map = &sync.Map{}
	core.Jobs.Map = &sync.Map{}
	core.NewBroker()
	core.NewSessions()
	core.ResetTransientTransportState()
}

func seedForwardDispatchSession(t testing.TB, sessionID, pipelineName, pipelineType string) *core.Session {
	t.Helper()

	listener := core.NewListener("test-listener", "127.0.0.1")
	core.Listeners.Add(listener)
	listener.AddPipeline(testPipelineProtobuf(pipelineName, pipelineType))

	sess, err := core.RegisterSession(&clientpb.RegisterSession{
		Type:       pipelineType,
		SessionId:  sessionID,
		RawId:      1,
		PipelineId: pipelineName,
		ListenerId: listener.Name,
		Target:     "127.0.0.1",
		RegisterData: &implantpb.Register{
			Name: "seed-artifact",
			Timer: &implantpb.Timer{
				Expression: "* * * * *",
			},
			Sysinfo: &implantpb.SysInfo{
				Os: &implantpb.Os{
					Name: "linux",
					Arch: "amd64",
				},
				Process: &implantpb.Process{
					Name: "seed",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterSession failed: %v", err)
	}
	sess.SetLastCheckin(time.Now().Unix())
	if err := sess.Save(); err != nil {
		t.Fatalf("session.Save failed: %v", err)
	}
	core.Sessions.Add(sess)
	return sess
}

func testPipelineProtobuf(name, pipelineType string) *clientpb.Pipeline {
	pb := &clientpb.Pipeline{
		Name:       name,
		ListenerId: "test-listener",
		Enable:     true,
		Type:       pipelineType,
		Parser:     "auto",
		Tls:        &clientpb.TLS{},
		Encryption: []*clientpb.Encryption{},
		Secure:     &clientpb.Secure{},
	}
	switch pipelineType {
	case consts.HTTPPipeline:
		pb.Body = &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Name:       name,
				ListenerId: "test-listener",
				Host:       "127.0.0.1",
				Port:       0,
				Params:     (&implanttypes.PipelineParams{}).String(),
			},
		}
	default:
		pb.Body = &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       name,
				ListenerId: "test-listener",
				Host:       "127.0.0.1",
				Port:       0,
			},
		}
	}
	return pb
}

func startForwardDispatchPipeline(t testing.TB, rpc pipelineRPCClient, pb *clientpb.Pipeline) func() {
	t.Helper()

	switch pb.Type {
	case consts.HTTPPipeline:
		pipeline, err := NewHttpPipeline(rpc, pb)
		if err != nil {
			t.Fatalf("NewHttpPipeline failed: %v", err)
		}
		if err := pipeline.Start(); err != nil {
			t.Fatalf("HTTPPipeline.Start failed: %v", err)
		}
		return func() { _ = pipeline.Close() }
	default:
		pipeline, err := NewTcpPipeline(rpc, pb)
		if err != nil {
			t.Fatalf("NewTcpPipeline failed: %v", err)
		}
		if err := pipeline.Start(); err != nil {
			t.Fatalf("TCPPipeline.Start failed: %v", err)
		}
		return func() { _ = pipeline.Close() }
	}
}

func TestForwardDispatchMissingConnectionCompletesTaskAcrossPollingPipelines(t *testing.T) {
	for _, tc := range []struct {
		name         string
		pipelineType string
	}{
		{name: "tcp", pipelineType: consts.TCPPipeline},
		{name: "http", pipelineType: consts.HTTPPipeline},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupForwardDispatchTest(t)

			pipelineName := "missing-connection-" + tc.name
			stream := newQueuedForwardStream()
			defer stream.close()
			rpc := &queuedPipelineRPCClient{stream: stream}
			closePipeline := startForwardDispatchPipeline(t, rpc, testPipelineProtobuf(pipelineName, tc.pipelineType))
			defer closePipeline()

			sessionID := "session-missing-connection-" + tc.name
			sess := seedForwardDispatchSession(t, sessionID, pipelineName, tc.pipelineType)
			task := sess.NewTask(consts.ModuleLs, 1)
			if err := db.AddTask(task.ToProtobuf()); err != nil {
				t.Fatalf("AddTask failed: %v", err)
			}

			spite := &implantpb.Spite{
				Name:   consts.ModuleLs,
				TaskId: task.Id,
				Async:  true,
				Body: &implantpb.Spite_Request{
					Request: &implantpb.Request{Name: consts.ModuleLs},
				},
			}
			stream.enqueue(&clientpb.SpiteRequest{
				Session: sess.ToProtobufLite(),
				Task:    task.ToProtobuf(),
				Spite:   spite,
			})

			deadline := time.Now().Add(500 * time.Millisecond)
			for time.Now().Before(deadline) {
				if task.Finished() {
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			t.Fatalf("task %s stayed running after %s forward dispatch lost the connection", task.TaskID(), tc.name)
		})
	}
}
