package listener_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	listenercmd "github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/spf13/cobra"
)

func TestStartPipelineCmdPropagatesStopFailure(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["pipe-a"] = &clientpb.Pipeline{Name: "pipe-a", Enable: true}
	h.Recorder.OnEmpty("StopPipeline", func(context.Context, any) (*clientpb.Empty, error) {
		return nil, errors.New("stop failed")
	})

	cmd := &cobra.Command{Use: "start"}
	cmd.Flags().String("cert-name", "", "")
	if err := cmd.Flags().Parse([]string{"pipe-a"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	err := listenercmd.StartPipelineCmd(cmd, h.Console)
	if err == nil || !strings.Contains(err.Error(), "stop failed") {
		t.Fatalf("StartPipelineCmd error = %v, want stop failure", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "StopPipeline" {
		t.Fatalf("calls = %#v, want only StopPipeline", calls)
	}
}

func TestStartPipelineCmdForwardsCertNameToStartRequest(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["pipe-b"] = &clientpb.Pipeline{Name: "pipe-b", Enable: false}

	cmd := &cobra.Command{Use: "start"}
	cmd.Flags().String("cert-name", "", "")
	if err := cmd.Flags().Parse([]string{"pipe-b"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if err := cmd.Flags().Set("cert-name", "cert-blue"); err != nil {
		t.Fatalf("Set(cert-name) failed: %v", err)
	}

	if err := listenercmd.StartPipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("StartPipelineCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "StartPipeline" {
		t.Fatalf("calls = %#v, want only StartPipeline", calls)
	}
	req, ok := calls[0].Request.(*clientpb.CtrlPipeline)
	if !ok {
		t.Fatalf("request type = %T, want *clientpb.CtrlPipeline", calls[0].Request)
	}
	if req.Name != "pipe-b" || req.CertName != "cert-blue" {
		t.Fatalf("start request = %#v, want pipe-b/cert-blue", req)
	}
}

func TestStartPipelineCmdUsesScopedCacheKey(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["listener-a:pipe-c"] = &clientpb.Pipeline{
		Name:       "pipe-c",
		ListenerId: "listener-a",
		Enable:     false,
	}

	cmd := &cobra.Command{Use: "start"}
	cmd.Flags().String("cert-name", "", "")
	if err := cmd.Flags().Parse([]string{"listener-a:pipe-c"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if err := listenercmd.StartPipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("StartPipelineCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "StartPipeline" {
		t.Fatalf("calls = %#v, want only StartPipeline", calls)
	}
	req, ok := calls[0].Request.(*clientpb.CtrlPipeline)
	if !ok {
		t.Fatalf("request type = %T, want *clientpb.CtrlPipeline", calls[0].Request)
	}
	if req.Name != "pipe-c" || req.ListenerId != "listener-a" {
		t.Fatalf("start request = %#v, want scoped pipe-c/listener-a", req)
	}
}

func TestStopPipelineCmdUsesScopedCacheKey(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["listener-b:pipe-d"] = &clientpb.Pipeline{
		Name:       "pipe-d",
		ListenerId: "listener-b",
	}

	cmd := &cobra.Command{Use: "stop"}
	if err := cmd.Flags().Parse([]string{"listener-b:pipe-d"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if err := listenercmd.StopPipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("StopPipelineCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "StopPipeline" {
		t.Fatalf("calls = %#v, want only StopPipeline", calls)
	}
	req, ok := calls[0].Request.(*clientpb.CtrlPipeline)
	if !ok {
		t.Fatalf("request type = %T, want *clientpb.CtrlPipeline", calls[0].Request)
	}
	if req.Name != "pipe-d" || req.ListenerId != "listener-b" {
		t.Fatalf("stop request = %#v, want scoped pipe-d/listener-b", req)
	}
}
