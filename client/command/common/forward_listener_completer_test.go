package common_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/spf13/cobra"
)

func TestForwardListenerIDCompleterUsesForwardListenersOnly(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Listeners["listener-a"] = &clientpb.Listener{Id: "listener-a"}
	h.Console.Listeners["listener-normal"] = &clientpb.Listener{Id: "listener-normal"}
	h.Recorder.OnForwardListenerStatuses("ListForwardListeners", func(_ context.Context, _ any) (*clientpb.ForwardListenerStatuses, error) {
		return &clientpb.ForwardListenerStatuses{Listeners: []*clientpb.ForwardListenerStatus{
			{ListenerId: "listener-a", Address: "127.0.0.1:5005", Active: true},
		}}, nil
	})

	values := completionValues(t, common.ForwardListenerIDCompleter(h.Console))

	if !hasCompletionValue(values, "listener-a") {
		t.Fatalf("completion values = %#v, want listener-a", values)
	}
	if hasCompletionValue(values, "listener-normal") {
		t.Fatalf("completion values = %#v, should not include normal listeners", values)
	}
}

func TestPipelineNameFlagCompleterUsesBareNamesAndFiltersListenerAndType(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["listener-a:shared"] = &clientpb.Pipeline{Name: "shared", ListenerId: "listener-a", Type: consts.HTTPPipeline}
	h.Console.Pipelines["listener-b:shared"] = &clientpb.Pipeline{Name: "shared", ListenerId: "listener-b", Type: consts.TCPPipeline}
	h.Console.Pipelines["listener-a:bind-only"] = &clientpb.Pipeline{Name: "bind-only", ListenerId: "listener-a", Type: consts.BindPipeline}

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("listener", "", "")
	action := common.PipelineNameFlagCompleter(h.Console, cmd, consts.HTTPPipeline, consts.TCPPipeline)
	values := completionValues(t, action)
	if countCompletionValue(values, "shared") != 1 {
		t.Fatalf("completion values = %#v, want one bare shared name", values)
	}
	if hasCompletionValue(values, "listener-a:shared") || hasCompletionValue(values, "bind-only") {
		t.Fatalf("completion values = %#v, want no scoped or unsupported values", values)
	}

	if err := cmd.Flags().Set("listener", "listener-b"); err != nil {
		t.Fatalf("set listener flag: %v", err)
	}
	values = completionValues(t, action)
	if countCompletionValue(values, "shared") != 1 {
		t.Fatalf("listener-filtered completion values = %#v, want shared", values)
	}
}

func completionValues(t testing.TB, action carapace.Action) []string {
	t.Helper()

	data, err := json.Marshal(action.Invoke(carapace.Context{}))
	if err != nil {
		t.Fatalf("marshal completion action failed: %v", err)
	}
	var decoded struct {
		Values []struct {
			Value string `json:"value"`
		} `json:"values"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode completion action failed: %v", err)
	}
	values := make([]string, 0, len(decoded.Values))
	for _, value := range decoded.Values {
		values = append(values, value.Value)
	}
	return values
}

func hasCompletionValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func countCompletionValue(values []string, want string) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}
