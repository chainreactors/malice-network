package common

import (
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/spf13/cobra"
)

func TestRegisterCommandUsesStandardCommandMetadata(t *testing.T) {
	con := &core.Console{
		CMDs:    map[string]*cobra.Command{},
		Helpers: map[string]*cobra.Command{},
	}
	root := &cobra.Command{Use: "implant"}
	sub := &cobra.Command{Use: "run", Annotations: map[string]string{"depend": "execute"}}
	cmd := &cobra.Command{Use: "example", Annotations: map[string]string{"os": "windows"}}
	cmd.AddCommand(sub)

	RegisterCommand(root, con, "example-mal", "mal", cmd)

	if cmd.Parent() != root {
		t.Fatal("command was not attached to the root")
	}
	if cmd.GroupID != "example-mal" || cmd.Annotations["menu"] != "implant" || cmd.Annotations["source"] != "mal" {
		t.Fatalf("command metadata = group %q, annotations %#v", cmd.GroupID, cmd.Annotations)
	}
	if con.CMDs[cmd.Name()] != cmd {
		t.Fatal("top-level command was not registered in console CMDs")
	}
	if con.Helpers["execute"] != sub {
		t.Fatal("dependency helper was not registered")
	}
}

func TestUnregisterCommandRemovesConsoleReferences(t *testing.T) {
	con := &core.Console{
		CMDs:    map[string]*cobra.Command{},
		Helpers: map[string]*cobra.Command{},
	}
	root := &cobra.Command{Use: "implant"}
	sub := &cobra.Command{Use: "run", Annotations: map[string]string{"depend": "execute"}}
	cmd := &cobra.Command{Use: "example"}
	cmd.AddCommand(sub)
	RegisterCommand(root, con, "example-mal", "mal", cmd)

	UnregisterCommand(root, con, cmd)

	if cmd.Parent() != nil {
		t.Fatal("command is still attached to the root")
	}
	if _, ok := con.CMDs[cmd.Name()]; ok {
		t.Fatal("top-level command remains in console CMDs")
	}
	if _, ok := con.Helpers["execute"]; ok {
		t.Fatal("dependency helper remains registered")
	}
}

func TestRegisterAndUnregisterPluginEventHooks(t *testing.T) {
	con := &core.Console{Server: &core.Server{ServerState: &iomclient.ServerState{
		EventHook: map[iomclient.EventCondition][]iomclient.OnEventFunc{},
	}}}
	condition := iomclient.EventCondition{Type: "startup-mal-event"}
	makeHook := func(result bool) iomclient.OnEventFunc {
		return func(*clientpb.Event) (bool, error) { return result, nil }
	}
	hook := makeHook(false)
	plug := &registerTestPlugin{
		manifest: &plugin.MalManiFest{Name: "event-mal"},
		events:   map[iomclient.EventCondition]iomclient.OnEventFunc{condition: hook},
	}

	RegisterPluginEventHooks(con, plug)
	if len(con.EventHook[condition]) != 1 {
		t.Fatalf("registered hooks = %d, want 1", len(con.EventHook[condition]))
	}
	otherHook := makeHook(true)
	otherPlug := &registerTestPlugin{
		manifest: &plugin.MalManiFest{Name: "other-event-mal"},
		events:   map[iomclient.EventCondition]iomclient.OnEventFunc{condition: otherHook},
	}
	RegisterPluginEventHooks(con, otherPlug)

	UnregisterPluginEventHooks(con, plug)
	if len(con.EventHook[condition]) != 1 {
		t.Fatalf("hooks after unregister = %d, want 1", len(con.EventHook[condition]))
	}
	remaining, err := con.EventHook[condition][0](nil)
	if err != nil || !remaining {
		t.Fatalf("remaining hook result = %v, err = %v", remaining, err)
	}

	UnregisterPluginEventHooks(con, otherPlug)
	if len(con.EventHook[condition]) != 0 {
		t.Fatalf("hooks after all unregister = %d, want 0", len(con.EventHook[condition]))
	}
}

type registerTestPlugin struct {
	manifest *plugin.MalManiFest
	events   map[iomclient.EventCondition]iomclient.OnEventFunc
}

func (p *registerTestPlugin) Run() error                    { return nil }
func (p *registerTestPlugin) Manifest() *plugin.MalManiFest { return p.manifest }
func (p *registerTestPlugin) Commands() plugin.Commands     { return nil }
func (p *registerTestPlugin) Destroy() error                { return nil }
func (p *registerTestPlugin) GetEvents() map[iomclient.EventCondition]iomclient.OnEventFunc {
	return p.events
}
