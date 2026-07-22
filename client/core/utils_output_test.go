package core

import (
	"fmt"
	"io"
	"strings"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func TestMergeCommandOutputs(t *testing.T) {
	tests := []struct {
		name           string
		bufferedOutput string
		cobraOutput    string
		want           string
	}{
		{
			name:           "logger output only",
			bufferedOutput: "logger output\n",
			want:           "logger output\n",
		},
		{
			name:        "Cobra output only",
			cobraOutput: "Usage:\n  demo\n",
			want:        "Usage:\n  demo\n",
		},
		{
			name:           "different output streams",
			bufferedOutput: "status",
			cobraOutput:    "result\nline two",
			want:           "status\nresult\nline two",
		},
		{
			name:           "existing line boundary",
			bufferedOutput: "status\n",
			cobraOutput:    "result",
			want:           "status\nresult",
		},
		{
			name:           "duplicate output",
			bufferedOutput: "same output\n",
			cobraOutput:    "same output",
			want:           "same output\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeCommandOutputs(tt.bufferedOutput, tt.cobraOutput); got != tt.want {
				t.Fatalf("mergeCommandOutputs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProgrammaticCommandPathsCaptureCobraOutput(t *testing.T) {
	con := &Console{
		Server: &Server{ServerState: &iomclient.ServerState{
			Client:       &clientpb.Client{Name: "test"},
			ActiveTarget: &iomclient.ActiveTarget{},
		}},
		Log:     iomclient.Log,
		CMDs:    make(map[string]*cobra.Command),
		Helpers: make(map[string]*cobra.Command),
	}
	con.NewConsole()

	root := &cobra.Command{Use: "client", SilenceErrors: true, SilenceUsage: true}
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.AddCommand(&cobra.Command{
		Use:   "demo",
		Short: "Demo command",
		Run:   func(*cobra.Command, []string) {},
	})
	con.App.Menu(consts.ClientMenu).Command = root
	con.App.SwitchMenu(consts.ClientMenu)

	tests := []struct {
		name string
		run  func() (string, error)
	}{
		{
			name: "RunCommand",
			run: func() (string, error) {
				return RunCommand(con, []string{"demo", "--help"})
			},
		},
		{
			name: "task-wait command",
			run: func() (string, error) {
				return executeCommand(con, "demo --help", "", consts.CalleeMCP)
			},
		},
		{
			name: "stream command",
			run: func() (string, error) {
				output, task, err := executeStreamCommand(con, "demo --help", "")
				if task != nil {
					return "", fmt.Errorf("executeStreamCommand task = %#v, want nil", task)
				}
				return output, err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := tt.run()
			if err != nil {
				t.Fatalf("command returned error: %v", err)
			}
			for _, want := range []string{"Demo command", "Usage:", "--help"} {
				if !strings.Contains(output, want) {
					t.Errorf("command output = %q, want it to contain %q", output, want)
				}
			}
			if strings.Count(output, "\n") < 2 {
				t.Errorf("command output = %q, want multiline help", output)
			}
		})
	}
}
