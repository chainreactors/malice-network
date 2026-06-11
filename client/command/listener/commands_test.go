package listener_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestListenerCommandConformance(t *testing.T) {
	testsupport.RunClientCases(t, []testsupport.CommandCase{
		{
			Name: "listener requests listener inventory",
			Argv: []string{consts.CommandListener},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnListeners("GetListeners", func(_ context.Context, _ any) (*clientpb.Listeners, error) {
					return &clientpb.Listeners{
						Listeners: []*clientpb.Listener{
							{Id: "listener-1", Ip: "127.0.0.1", Active: true},
						},
					}, nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.MustSingleCall[*clientpb.Empty](t, h, "GetListeners")
			},
		},
		{
			Name: "listener propagates rpc errors",
			Argv: []string{consts.CommandListener},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnListeners("GetListeners", func(_ context.Context, _ any) (*clientpb.Listeners, error) {
					return nil, errors.New("listener list failed")
				})
			},
			WantErr: "listener list failed",
		},
		{
			Name: "job requests pipeline jobs",
			Argv: []string{consts.CommandJob},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnPipelines("ListJobs", func(_ context.Context, _ any) (*clientpb.Pipelines, error) {
					return &clientpb.Pipelines{
						Pipelines: []*clientpb.Pipeline{
							{
								Name:       "tcp-job",
								ListenerId: "listener-1",
								Ip:         "0.0.0.0",
								Body: &clientpb.Pipeline_Tcp{
									Tcp: &clientpb.TCPPipeline{Port: 4444},
								},
							},
							{
								Name:       "web-job",
								ListenerId: "listener-2",
								Ip:         "127.0.0.1",
								Body: &clientpb.Pipeline_Web{
									Web: &clientpb.Website{Port: 8443},
								},
							},
							{
								Name:       "rem-job",
								ListenerId: "listener-3",
								Ip:         "10.0.0.1",
								Type:       "rem",
							},
						},
					}, nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.MustSingleCall[*clientpb.Empty](t, h, "ListJobs")
			},
		},
		{
			Name: "job propagates rpc errors",
			Argv: []string{consts.CommandJob},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnPipelines("ListJobs", func(_ context.Context, _ any) (*clientpb.Pipelines, error) {
					return nil, errors.New("job list failed")
				})
			},
			WantErr: "job list failed",
		},
	})
}
