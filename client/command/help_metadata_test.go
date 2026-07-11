package command_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/chainreactors/malice-network/client/command"
	buildcmd "github.com/chainreactors/malice-network/client/command/build"
	certcmd "github.com/chainreactors/malice-network/client/command/cert"
	listenercmd "github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	websitecmd "github.com/chainreactors/malice-network/client/command/website"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func TestComplexClientCommandsExposeActionableHelp(t *testing.T) {
	con := &core.Console{}
	website := mustCommandPath(t, websitecmd.Commands(con), "website")
	artifact := mustCommandPath(t, buildcmd.Commands(con), "artifact")
	cert := mustCommandPath(t, certcmd.Commands(con), "cert")
	listener := mustCommandPath(t, listenercmd.Commands(con), "listener")
	pipeline := mustCommandPath(t, listenercmd.Commands(con), "pipeline")

	tests := []struct {
		name         string
		cmd          *cobra.Command
		wantUse      string
		wantLong     []string
		wantExamples []string
	}{
		{
			name:         "website cert",
			cmd:          mustSubcommandPath(t, website, "cert"),
			wantUse:      "cert [website_name]",
			wantLong:     []string{"exactly one", "--cert-name", "--cert/--key", "--generate", "--disable"},
			wantExamples: []string{"website cert site-a --cert-name ZANY_PIN", "website cert site-a --generate", "website cert site-a --disable"},
		},
		{
			name:         "website tls",
			cmd:          mustSubcommandPath(t, website, "tls"),
			wantUse:      "tls [website_name]",
			wantLong:     []string{"exactly one", "--save-cert-name"},
			wantExamples: []string{"website tls site-a --cert-name ZANY_PIN"},
		},
		{
			name:         "website route add",
			cmd:          mustSubcommandPath(t, website, "route", "add"),
			wantUse:      "add [file_path]",
			wantLong:     []string{"--artifact", "file_path"},
			wantExamples: []string{"website route add /path/to/index.html", "website route add --artifact beacon"},
		},
		{
			name:         "artifact publish",
			cmd:          mustSubcommandPath(t, artifact, "publish"),
			wantUse:      "publish [artifact_name]",
			wantLong:     []string{"--website", "--path", "--format"},
			wantExamples: []string{"artifact publish beacon --website payloads --path /beacon.bin"},
		},
		{
			name:         "artifact prune",
			cmd:          mustSubcommandPath(t, artifact, "prune"),
			wantUse:      "prune",
			wantLong:     []string{"--failed", "--older-than"},
			wantExamples: []string{"artifact prune --failed", "artifact prune --older-than 720h"},
		},
		{
			name:         "certificate prune",
			cmd:          mustSubcommandPath(t, cert, "prune"),
			wantUse:      "prune",
			wantLong:     []string{"expired", "--expired"},
			wantExamples: []string{"cert prune --expired"},
		},
		{
			name:         "listener forward disconnect",
			cmd:          mustSubcommandPath(t, listener, "forward", "disconnect"),
			wantUse:      "disconnect [listener_id]",
			wantLong:     []string{"forward listener"},
			wantExamples: []string{"listener forward disconnect listener-a"},
		},
		{
			name:         "pipeline update",
			cmd:          mustSubcommandPath(t, pipeline, "update"),
			wantUse:      "update [pipeline_name]",
			wantLong:     []string{"--enable", "--disable", "--cert-name", "--parser"},
			wantExamples: []string{"pipeline update tcp-main --cert-name web-cert"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Use != tt.wantUse {
				t.Fatalf("Use = %q, want %q", tt.cmd.Use, tt.wantUse)
			}
			for _, want := range tt.wantLong {
				if !strings.Contains(tt.cmd.Long, want) {
					t.Errorf("Long help does not contain %q:\n%s", want, tt.cmd.Long)
				}
			}
			for _, want := range tt.wantExamples {
				if !strings.Contains(tt.cmd.Example, want) {
					t.Errorf("Examples do not contain %q:\n%s", want, tt.cmd.Example)
				}
			}
		})
	}
}

func TestClientRootLetsConsoleRenderExecutionErrorsOnce(t *testing.T) {
	h := testsupport.NewHarness(t)
	roots := map[string]*cobra.Command{
		"client":  command.BindClientsCommands(h.Console)(),
		"implant": command.BindImplantCommands(h.Console)(),
	}
	for name, root := range roots {
		if !root.SilenceErrors {
			t.Errorf("%s root should silence Cobra errors so the console renders them once", name)
		}
	}
}

func TestBuiltInRunnableCommandsHaveDetailedHelp(t *testing.T) {
	h := testsupport.NewHarness(t)
	roots := []*cobra.Command{
		command.BindClientsCommands(h.Console)(),
		command.BindImplantCommands(h.Console)(),
	}

	var problems []string
	seen := make(map[string]struct{})
	for _, root := range roots {
		walkBuiltInCommands(root, "golang", func(cmd *cobra.Command) {
			if cmd == root || cmd.Hidden || !cmd.Runnable() || cmd.Name() == "help" {
				return
			}
			path := strings.TrimPrefix(cmd.CommandPath(), root.Name()+" ")
			if _, ok := seen[path]; ok {
				return
			}
			seen[path] = struct{}{}
			if strings.TrimSpace(cmd.Short) == "" {
				problems = append(problems, fmt.Sprintf("%s: missing Short", cmd.CommandPath()))
			}
			if strings.TrimSpace(cmd.Long) == "" {
				problems = append(problems, fmt.Sprintf("%s: missing Long", cmd.CommandPath()))
			}
			if strings.TrimSpace(cmd.Example) == "" {
				problems = append(problems, fmt.Sprintf("%s: missing Example", cmd.CommandPath()))
			}
			if cmd.Args != nil && cmd.Args(cmd, nil) != nil && !strings.ContainsAny(cmd.Use, "[<") {
				problems = append(problems, fmt.Sprintf("%s: required positional argument is absent from Use %q", cmd.CommandPath(), cmd.Use))
			}
		})
	}

	if len(problems) != 0 {
		sort.Strings(problems)
		t.Fatalf("built-in command help audit failed:\n%s", strings.Join(problems, "\n"))
	}
}

func walkBuiltInCommands(cmd *cobra.Command, source string, visit func(*cobra.Command)) {
	if strings.HasPrefix(cmd.Name(), "_carapace") {
		return
	}
	if ownSource := cmd.Annotations["source"]; ownSource != "" {
		source = ownSource
	}
	if source != "golang" {
		return
	}
	visit(cmd)
	for _, child := range cmd.Commands() {
		walkBuiltInCommands(child, source, visit)
	}
}

func mustCommandPath(t testing.TB, commands []*cobra.Command, name string) *cobra.Command {
	t.Helper()
	for _, cmd := range commands {
		if cmd.Name() == name {
			return cmd
		}
	}
	t.Fatalf("command %q not found", name)
	return nil
}

func mustSubcommandPath(t testing.TB, root *cobra.Command, path ...string) *cobra.Command {
	t.Helper()
	cmd, _, err := root.Find(path)
	if err != nil {
		t.Fatalf("find %q below %q: %v", strings.Join(path, " "), root.Name(), err)
	}
	return cmd
}
